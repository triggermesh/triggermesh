/*
Copyright 2022 TriggerMesh Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package awssqssource

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/health"
)

const (
	logfieldMsgID  = "msgID"
	logfieldMsgIDs = "msgIDs"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	ARN string `envconfig:"ARN" required:"true"`

	// Assume this IAM Role when access keys provided.
	AssumeIamRole string `envconfig:"AWS_ASSUME_ROLE_ARN"`

	// Name of a message processor which takes care of converting SQS
	// messages to CloudEvents.
	//
	// Supported values: [ default s3 eventbridge ]
	MessageProcessor string `envconfig:"SQS_MESSAGE_PROCESSOR" default:"default"`

	// https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html
	// Visibility timeout to set on all messages received by this event source.
	// https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html
	VisibilityTimeout *time.Duration `envconfig:"SQS_VISIBILITY_TIMEOUT"`

	// Allows overriding common CloudEvents attributes.
	CEOverrideSource string `envconfig:"CE_SOURCE"`
	CEOverrideType   string `envconfig:"CE_TYPE"`

	// The environment variables below aren't read from the envConfig struct
	// by the AWS SDK, but rather directly using os.Getenv().
	// They are nevertheless listed here for documentation purposes.
	_ string `envconfig:"AWS_ACCESS_KEY_ID"`
	_ string `envconfig:"AWS_SECRET_ACCESS_KEY"`

	MaxBatchSize         string `envconfig:"AWS_SQS_MAX_BATCH_SIZE" default:"10"`
	SendBatchedResponse  string `envconfig:"AWS_SQS_SEND_BATCH_RESPONSE" default:"false"`
	OnFailedPollWaitSecs string `envconfig:"AWS_SQS_POLL_FAILED_WAIT_TIME" default:"2"`
	WaitTimeSeconds      string `envconfig:"AWS_SQS_WAIT_TIME_SECONDS" default:"3"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger *zap.SugaredLogger

	mt *pkgadapter.MetricTag
	sr *statsReporter

	sqsClient sqsiface.SQSAPI
	ceClient  cloudevents.Client

	arn arn.ARN

	msgPrcsr MessageProcessor

	visibilityTimeoutSeconds *int64

	processQueue chan *sqs.Message
	deleteQueue  chan *sqs.Message

	deletePeriod time.Duration

	maxBatchSize         string
	sendBatchedResponse  string
	onFailedPollWaitSecs string
	waitTimeSeconds      string
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mustRegisterStatsView()

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.AWSSQSSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	arn := common.MustParseARN(env.ARN)

	var msgPrcsr MessageProcessor
	switch env.MessageProcessor {
	case "s3":
		msgPrcsr = &s3MessageProcessor{
			ceSourceFallback: arn.String(),
		}
	case "eventbridge":
		msgPrcsr = &eventbridgeMessageProcessor{
			ceSource:         env.CEOverrideSource,
			ceSourceFallback: arn.String(),
		}
	case "default":
		msgPrcsr = &defaultMessageProcessor{
			ceSource: arn.String(),
		}
	default:
		panic("Unsupported message processor " + strconv.Quote(env.MessageProcessor))
	}

	var visibilityTimeoutSeconds *int64
	if vt := env.VisibilityTimeout; vt != nil {
		if *vt < 0 || *vt > 12*time.Hour {
			logger.Warn("Ignoring out of bounds visibility timeout (", *vt, ")")
		} else {
			vts := durationInSeconds(*vt)
			visibilityTimeoutSeconds = &vts
		}
	}

	sess := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(arn.Region).
		WithEndpointResolver(common.EndpointResolver(arn.Partition)),
	))

	config := &aws.Config{}
	if env.AssumeIamRole != "" {
		config.Credentials = stscreds.NewCredentials(sess, env.AssumeIamRole)
	}

	// allocate generous buffer sizes to limit blocking on surges of new
	// messages coming from receivers
	const batchSizePerProc = 9
	queueBufferSizeProcess := maxReceiveMsgBatchSize * runtime.GOMAXPROCS(-1) * batchSizePerProc
	queueBufferSizeDelete := queueBufferSizeProcess

	sr := mustNewStatsReporter(mt)
	sr.reportQueueCapacityProcess(queueBufferSizeProcess)
	sr.reportQueueCapacityDelete(queueBufferSizeDelete)

	maxBatchSize := env.MaxBatchSize
	sendBatchedResponse := env.SendBatchedResponse
	onFailedPollWaitSecs := env.OnFailedPollWaitSecs
	waitTimeSeconds := env.WaitTimeSeconds

	return &adapter{
		logger: logger,

		mt: mt,
		sr: sr,

		sqsClient: sqs.New(sess, config),
		ceClient:  ceClient,

		arn: arn,

		msgPrcsr: msgPrcsr,

		visibilityTimeoutSeconds: visibilityTimeoutSeconds,

		processQueue: make(chan *sqs.Message, queueBufferSizeProcess),
		deleteQueue:  make(chan *sqs.Message, queueBufferSizeDelete),

		deletePeriod: maxDeleteMsgPeriod,

		maxBatchSize:         maxBatchSize,
		sendBatchedResponse:  sendBatchedResponse,
		onFailedPollWaitSecs: onFailedPollWaitSecs,
		waitTimeSeconds:      waitTimeSeconds,
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	url, err := a.queueLookup(a.arn.Resource)
	if err != nil {
		a.logger.Errorw("Unable to find URL of SQS queue "+a.arn.Resource, zap.Error(err))
		return err
	}

	health.MarkReady()

	queueURL := *url.QueueUrl
	a.logger.Infof("Listening to SQS queue at URL: %s", queueURL)

	logger := logging.FromContext(ctx)

	logger.Info("Starting with config: ", zap.Any("adapter", a))

	msgCtx, cancel := context.WithCancel(pkgadapter.ContextWithMetricTag(ctx, a.mt))
	defer cancel()

	var wg sync.WaitGroup

	// This event source spends most of its time waiting for the network,
	// so we can run more than one of each receiver|processor|deleter for
	// each available thread.

	// TODO(antoineco): spawn and terminate receivers dynamically
	// based on the current amount of messages being processed to
	// optimize costs generated by ReceiveMessage API requests.
	// https://github.com/triggermesh/triggermesh/issues/227
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.pollLoop(msgCtx, queueURL)
	}()

	<-ctx.Done()
	cancel()

	a.logger.Info("Waiting for message handlers to terminate")
	wg.Wait()

	return nil
}

func (a *adapter) getDeleteMessageEntries(sqsMessages []*sqs.Message) (Entries []*sqs.DeleteMessageBatchRequestEntry) {
	var list []*sqs.DeleteMessageBatchRequestEntry
	for _, message := range sqsMessages {
		list = append(list, &sqs.DeleteMessageBatchRequestEntry{
			Id:            message.MessageId,
			ReceiptHandle: message.ReceiptHandle,
		})
	}
	return list
}

// pollLoop continuously polls from the given SQS queue until stopCh
// emits an element.  The
func (a *adapter) pollLoop(ctx context.Context, queueURL string) error {

	logger := logging.FromContext(ctx)

	maxBatchSize, err := strconv.ParseInt(a.maxBatchSize, 10, 64)
	if err != nil {
		logger.Info("Could not Find or convert maxBatchSize from string to int. Defaulting", zap.Error(err))
	}
	sendBatchedResponse, err := strconv.ParseBool(a.sendBatchedResponse)
	if err != nil {
		logger.Info("Could not Find or convert sendBatchedResponse from string to bool, Defaulting", zap.Error(err))
	}
	onFailedPollWaitSecs, err := strconv.ParseInt(a.onFailedPollWaitSecs, 10, 0)
	if err != nil {
		logger.Info("Could not Find or convert onFailedPollWaitSecs from string to time.Duration. Defaulting ", zap.Error(err))
	}
	waitTimeSeconds, err := strconv.ParseInt(a.waitTimeSeconds, 10, 64)
	if err != nil {
		logger.Info("Could not Find or convert waitTimeSeconds from string to int. Defaulting ", zap.Error(err))
	}

	logger.Infof("value from configs: MaxBatchSize: %d, SendBatchedResponse: %b, OnFailedPollWaitSecs: %d, WaitTimeSeconds: %d", maxBatchSize, sendBatchedResponse, onFailedPollWaitSecs, waitTimeSeconds)

	for {
		messages, err := poll(ctx, a.sqsClient, queueURL, maxBatchSize, waitTimeSeconds)
		if err != nil {
			logger.Warn("Failed to poll from SQS queue", zap.Error(err))
			time.Sleep(time.Duration(onFailedPollWaitSecs) * time.Second)
			continue
		}

		if sendBatchedResponse && len(messages) > 0 {
			a.receiveMessages(ctx, messages, queueURL, func() {
				_, err = a.sqsClient.DeleteMessageBatch(&sqs.DeleteMessageBatchInput{
					QueueUrl: &queueURL,
					Entries:  a.getDeleteMessageEntries(messages),
				})
				if err != nil {
					// the only consequence is that the message will
					// get redelivered later, given that SQS is
					// at-least-once delivery. That should be
					// acceptable as "normal operation"
					logger.Error("Failed to delete messages", zap.Error(err))
				}
			})
		} else {
			for _, m := range messages {
				a.receiveMessage(ctx, m, queueURL, func() {
					_, err = a.sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
						QueueUrl:      &queueURL,
						ReceiptHandle: m.ReceiptHandle,
					})
					if err != nil {
						// the only consequence is that the message will
						// get redelivered later, given that SQS is
						// at-least-once delivery. That should be
						// acceptable as "normal operation"
						logger.Error("Failed to delete message", zap.Error(err))
					}
				})
			}

		}
	}
}

// receiveMessage handles an incoming message from the AWS SQS queue,
// and forwards it to a Sink, calling `ack()` when the forwarding is
// successful.
func (a *adapter) receiveMessage(ctx context.Context, m *sqs.Message, queueURL string, ack func()) {
	logger := logging.FromContext(ctx)
	logger.Debugw("Received message from SQS:", zap.Any("message", m))

	//ctx = cloudevents.ContextWithTarget(ctx, a.SinkURI)

	err := a.postMessage(ctx, logger, m, queueURL)
	if err != nil {
		logger.Infof("Event delivery failed: %s", err)
	} else {
		logger.Debug("Message successfully posted to Sink")
		ack()
	}
}

// receiveMessages handles an incoming list of message from the AWS SQS queue,
// and forwards it to a Sink, calling `ack()` when the forwarding is
// successful.
func (a *adapter) receiveMessages(ctx context.Context, messages []*sqs.Message, queueURL string, ack func()) {
	logger := logging.FromContext(ctx)
	logger.Debugw("Received messages from SQS:", zap.Any("messagesLength", len(messages)))

	//ctx = cloudevents.ContextWithTarget(ctx, a.SinkURI)

	err := a.postMessages(ctx, logger, messages, queueURL)
	if err != nil {
		logger.Infof("Event delivery failed: %s", err)
	} else {
		logger.Debug("Message successfully posted to Sink")
		ack()
	}
}

func (a *adapter) makeEvent(m *sqs.Message, queueURL string) (*cloudevents.Event, error) {
	timestamp, err := strconv.ParseInt(*m.Attributes["SentTimestamp"], 10, 64)
	if err == nil {
		//Convert to nanoseconds as sqs SentTimestamp is millisecond
		timestamp = timestamp * int64(1000000)
	} else {
		timestamp = time.Now().UnixNano()
	}

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetID(*m.MessageId)
	event.SetType(v1alpha1.AWSEventType(sqs.ServiceName, v1alpha1.AWSSQSGenericEventType))
	event.SetSource(cloudevents.ParseURIRef(queueURL).String())
	event.SetTime(time.Unix(0, timestamp))

	if err := event.SetData(cloudevents.ApplicationJSON, m); err != nil {
		return nil, err
	}
	return &event, nil
}

func (a *adapter) makeBatchEvent(messages []*sqs.Message, queueURL string) (*cloudevents.Event, error) {
	timestamp, err := strconv.ParseInt(*messages[0].Attributes["SentTimestamp"], 10, 64)
	if err == nil {
		//Convert to nanoseconds as sqs SentTimestamp is millisecond
		timestamp = timestamp * int64(1000000)
	} else {
		timestamp = time.Now().UnixNano()
	}

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetID(*messages[0].MessageId)
	event.SetType(v1alpha1.AWSEventType(sqs.ServiceName, v1alpha1.AWSSQSGenericEventType))
	event.SetSource(cloudevents.ParseURIRef(queueURL).String())
	event.SetTime(time.Unix(0, timestamp))

	if err := event.SetData(cloudevents.ApplicationJSON, messages); err != nil {
		return nil, err
	}
	return &event, nil
}

// postMessage sends an SQS event to the SinkURI
func (a *adapter) postMessage(ctx context.Context, logger *zap.SugaredLogger, m *sqs.Message, queueURL string) error {
	event, err := a.makeEvent(m, queueURL)

	if err != nil {
		logger.Error("Cloud Event creation error", zap.Error(err))
		return err
	}
	if result := a.ceClient.Send(ctx, *event); !cloudevents.IsACK(result) {
		logger.Error("Cloud Event delivery error", zap.Error(result))
		return result
	}
	return nil
}

// postMessages sends an array of SQS events to the SinkURI
func (a *adapter) postMessages(ctx context.Context, logger *zap.SugaredLogger, messages []*sqs.Message, queueURL string) error {
	event, err := a.makeBatchEvent(messages, queueURL)

	if err != nil {
		logger.Error("Cloud Event creation error", zap.Error(err))
		return err
	}
	if result := a.ceClient.Send(ctx, *event); !cloudevents.IsACK(result) {
		logger.Error("Cloud Event delivery error", zap.Error(result))
		return result
	}
	return nil
}

// poll reads messages from the queue in batches of a given maximum size.
func poll(ctx context.Context, q sqsiface.SQSAPI, url string, maxBatchSize int64, waitTimeSeconds int64) ([]*sqs.Message, error) {

	result, err := q.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl: &url,
		// Maximum size of the batch of messages returned from the poll.
		MaxNumberOfMessages: aws.Int64(maxBatchSize),
		// Controls the maximum time to wait in the poll performed with
		// ReceiveMessageWithContext.  If there are no messages in the
		// given secs, the call times out and returns control to us.
		// TODO: expose this as ENV variable
		WaitTimeSeconds: aws.Int64(waitTimeSeconds),
	})

	if err != nil {
		return []*sqs.Message{}, err
	}

	return result.Messages, nil
}

// durationInSeconds returns a duration as a number of seconds truncated
// towards zero.
func durationInSeconds(d time.Duration) int64 {
	// converting a floating-point number to an integer discards
	// the fraction (truncation towards zero)
	return int64(d.Seconds())
}

// queueLookup finds the URL for a given queue name in the user's account.
// Needs to be an exact match to queue name and queue must be unique name in the AWS account.
func (a *adapter) queueLookup(queueName string) (*sqs.GetQueueUrlOutput, error) {
	return a.sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})
}

// prettifyBatchResultErrors returns a pretty string representing a list of
// batch failures.
func prettifyBatchResultErrors(errs []*sqs.BatchResultErrorEntry) string {
	if len(errs) == 0 {
		return ""
	}

	var errStr strings.Builder

	errStr.WriteByte('[')

	for i, f := range errs {
		errStr.WriteString(f.String())
		if i+1 < len(errs) {
			errStr.WriteByte(',')
		}
	}

	errStr.WriteByte(']')

	return errStr.String()
}
