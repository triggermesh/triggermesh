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

package awssqstarget

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// NewTarget Adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.AWSSQSTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	a := MustParseARN(env.AwsTargetArn)

	sess := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(a.Region).
		WithMaxRetries(5)))

	config := &aws.Config{}
	if env.AssumeIamRole != "" {
		config.Credentials = stscreds.NewCredentials(sess, env.AssumeIamRole)
	}

	return &adapter{
		awsArnString:     env.AwsTargetArn,
		awsArn:           a,
		discardCEContext: env.DiscardCEContext,
		sqsClient:        sqs.New(sess, config),
		messageGroupID:   env.MessageGroupID,

		ceClient: ceClient,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

type adapter struct {
	awsArnString   string
	awsArn         arn.ARN
	sqsClient      *sqs.SQS
	messageGroupID string

	discardCEContext bool
	ceClient         cloudevents.Client
	logger           *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting AWS SQS Target adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

// Parse and send the aws event
func (a *adapter) dispatch(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var msg []byte

	if a.discardCEContext {
		msg = event.Data()
	} else {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			return a.reportError("Error marshalling CloudEvent", err)
		}
		msg = jsonEvent
	}

	// The SendMessageInput only accepts a URL for publishing messages. This can be extracted from the ARN
	url := "https://" + a.awsArn.Service + "." + a.awsArn.Region + ".amazonaws.com/" + a.awsArn.AccountID + "/" + a.awsArn.Resource

	var err error
	var result *sqs.SendMessageOutput
	if strings.HasSuffix(url, ".fifo") {
		dedupID := event.ID() + ";" + event.Source()
		result, err = a.sqsClient.SendMessage(&sqs.SendMessageInput{
			MessageBody:            aws.String(string(msg)),
			QueueUrl:               &url,
			MessageGroupId:         &a.messageGroupID,
			MessageDeduplicationId: &dedupID,
		})
	} else {
		result, err = a.sqsClient.SendMessage(&sqs.SendMessageInput{
			MessageBody: aws.String(string(msg)),
			QueueUrl:    &url,
		})
	}

	if err != nil {
		return a.reportError("error publishing to sqs", err)
	}

	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)
	err = responseEvent.SetData(cloudevents.ApplicationJSON, result.GoString())
	if err != nil {
		return a.reportError("error generating response event", err)
	}

	responseEvent.SetType("io.triggermesh.targets.aws.sqs.result")
	responseEvent.SetSource(a.awsArnString)

	return &responseEvent, cloudevents.ResultACK
}

func (a *adapter) reportError(msg string, err error) (*cloudevents.Event, cloudevents.Result) {
	a.logger.Errorw(msg, zap.Error(err))
	return nil, cloudevents.NewHTTPResult(http.StatusInternalServerError, msg)
}
