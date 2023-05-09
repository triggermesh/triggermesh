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

package awsdynamodbtarget

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// NewTarget Adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.AWSDynamoDBTargetResource.String(),
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

	var dynamodbTable string
	if a.Service == dynamodb.ServiceName {
		dynamodbTable = MustParseDynamoDBResource(a.Resource)
	}

	return &adapter{
		awsArnString:         env.AwsTargetArn,
		awsArn:               a,
		awsDynamoDBTableName: dynamodbTable,
		dynamoDBClient:       dynamodb.New(sess, config),

		discardCEContext: env.DiscardCEContext,
		ceClient:         ceClient,
		logger:           logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

type adapter struct {
	awsArnString         string
	awsArn               arn.ARN
	awsDynamoDBTableName string
	dynamoDBClient       *dynamodb.DynamoDB

	discardCEContext bool
	ceClient         cloudevents.Client
	logger           *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting AWS DynamoDB Target adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

// Parse and send the aws event
func (a *adapter) dispatch(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var eventJSONMap map[string]interface{}

	if a.discardCEContext {
		if err := event.DataAs(&eventJSONMap); err != nil {
			return a.reportError("Error deserializing event data to map", err)
		}
	} else {
		b, err := json.Marshal(event)
		if err != nil {
			return a.reportError("Error serializing event to JSON", err)
		}
		if err := json.Unmarshal(b, &eventJSONMap); err != nil {
			return a.reportError("Error deserializing JSON event to map", err)
		}
	}

	av, err := dynamodbattribute.MarshalMap(eventJSONMap)
	if err != nil {
		return a.reportError("Error marshalling attribute", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: &a.awsDynamoDBTableName,
	}

	resp, err := a.dynamoDBClient.PutItem(input)
	if err != nil {
		return a.reportError("Error invoking DynamoDB", err)
	}

	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)
	err = responseEvent.SetData(cloudevents.ApplicationJSON, resp)
	if err != nil {
		return a.reportError("error generating response event", err)
	}

	responseEvent.SetType(v1alpha1.EventTypeAWSDynamoDBResult)
	responseEvent.SetSource(a.awsArnString)
	return &responseEvent, cloudevents.ResultACK
}

func (a *adapter) reportError(msg string, err error) (*cloudevents.Event, cloudevents.Result) {
	a.logger.Errorw(msg, zap.Error(err))
	return nil, cloudevents.NewHTTPResult(http.StatusInternalServerError, msg)
}
