/*
Copyright 2021 TriggerMesh Inc.

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

package awssnstarget

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"
)

// NewTarget Adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	a := MustParseARN(env.AwsTargetArn)

	snsSession := session.Must(session.NewSession(
		env.GetAwsConfig().
			WithRegion(a.Region).
			WithMaxRetries(5)))

	return &adapter{
		awsArnString: env.AwsTargetArn,
		awsArn:       a,
		snsClient:    sns.New(snsSession),

		discardCEContext: env.DiscardCEContext,
		ceClient:         ceClient,
		logger:           logger,
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

type adapter struct {
	awsArnString string
	awsArn       arn.ARN
	snsClient    *sns.SNS

	discardCEContext bool
	ceClient         cloudevents.Client
	logger           *zap.SugaredLogger
}

func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting AWS SNS adapter")
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

	result, err := a.snsClient.Publish(&sns.PublishInput{
		Message:  aws.String(string(msg)),
		TopicArn: &a.awsArnString,
	})

	if err != nil {
		return a.reportError("error publishing to sns", err)
	}

	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)
	err = responseEvent.SetData(cloudevents.ApplicationJSON, result.GoString())
	if err != nil {
		return a.reportError("error generating response event", err)
	}

	responseEvent.SetType("io.triggermesh.targets.aws.sns.result")
	responseEvent.SetSource(a.awsArnString)

	return &responseEvent, cloudevents.ResultACK
}

func (a *adapter) reportError(msg string, err error) (*cloudevents.Event, cloudevents.Result) {
	a.logger.Errorw(msg, zap.Error(err))
	return nil, cloudevents.NewHTTPResult(http.StatusInternalServerError, msg)
}
