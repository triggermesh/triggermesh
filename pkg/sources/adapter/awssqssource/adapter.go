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

	msgCtx, cancel := context.WithCancel(pkgadapter.ContextWithMetricTag(ctx, a.mt))
	defer cancel()

	var wg sync.WaitGroup

	// This event source spends most of its time waiting for the network,
	// so we can run more than one of each receiver|processor|deleter for
	// each available thread.
	const instancesPerProc = 3

	for i := 0; i < runtime.GOMAXPROCS(-1)*instancesPerProc; i++ {
		// TODO(antoineco): spawn and terminate receivers dynamically
		// based on the current amount of messages being processed to
		// optimize costs generated by ReceiveMessage API requests.
		// https://github.com/triggermesh/triggermesh/issues/227
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runMessagesReceiver(msgCtx, queueURL)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runMessagesProcessor(msgCtx)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runMessagesDeleter(msgCtx, queueURL)
		}()
	}

	<-ctx.Done()
	cancel()

	a.logger.Debug("Waiting for message handlers to terminate")
	wg.Wait()

	return nil
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

// durationInSeconds returns a duration as a number of seconds truncated
// towards zero.
func durationInSeconds(d time.Duration) int64 {
	// converting a floating-point number to an integer discards
	// the fraction (truncation towards zero)
	return int64(d.Seconds())
}
