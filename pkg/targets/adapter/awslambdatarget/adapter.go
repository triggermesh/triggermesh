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

package awslambdatarget

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"
)

// Adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	config := env.GetAwsConfig()
	logger := logging.FromContext(ctx)

	a := MustParseARN(env.AwsTargetArn)

	config = config.WithRegion(a.Region)

	return &awsAdapter{
		config:           config, // define configuration for the aws client
		awsArnString:     env.AwsTargetArn,
		awsArn:           a,
		discardCEContext: env.DiscardCEContext,
		ceClient:         ceClient,

		logger: logger,
	}
}

var _ pkgadapter.Adapter = (*awsAdapter)(nil)

type awsAdapter struct {
	awsArnString string
	awsArn       arn.ARN
	config       *aws.Config
	session      *session.Session
	lda          *lambda.Lambda

	discardCEContext bool

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

func (a *awsAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting AWS Lambda adapter")
	s := session.Must(session.NewSession(a.config))
	a.session = s

	a.lda = lambda.New(s)

	if err := a.ceClient.StartReceiver(ctx, a.dispatch); err != nil {
		return err
	}
	return nil
}

// Parse and send the aws event
func (a *awsAdapter) dispatch(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var fnPayload []byte

	if a.discardCEContext {
		fnPayload = event.Data()
	} else {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			return a.reportError("Error marshalling CloudEvent", err)
		}
		fnPayload = jsonEvent
	}

	input := &lambda.InvokeInput{
		Payload:      fnPayload,
		FunctionName: &a.awsArnString,
	}
	out, err := a.lda.Invoke(input)
	if err != nil {
		return a.reportError("error invoking lambda", err)
	}

	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)
	err = responseEvent.SetData(cloudevents.ApplicationJSON, out.Payload)
	if err != nil {
		return a.reportError("error generating response event", err)
	}

	responseEvent.SetType("io.triggermesh.targets.aws.lambda.result")
	responseEvent.SetSource(a.awsArnString)

	return &responseEvent, cloudevents.ResultACK
}

func (a *awsAdapter) reportError(msg string, err error) (*cloudevents.Event, cloudevents.Result) {
	a.logger.Errorw(msg, zap.Error(err))
	return nil, cloudevents.NewHTTPResult(http.StatusInternalServerError, msg)
}
