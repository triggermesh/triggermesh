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

package awsdynamodbtarget

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// NewTarget Adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	config := env.GetAwsConfig()
	logger := logging.FromContext(ctx)

	a := MustParseARN(env.AwsTargetArn)

	config = config.WithRegion(a.Region)

	var dynamodbTable string
	if a.Service == dynamodb.ServiceName {
		dynamodbTable = MustParseDynamoDBResource(a.Resource)
	}

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeAWSDynamoDBResult),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &adapter{
		config:               config, // define configuration for the aws client
		awsArnString:         env.AwsTargetArn,
		awsArn:               a,
		awsDynamoDBTableName: dynamodbTable,

		replier:          replier,
		discardCEContext: env.DiscardCEContext,
		ceClient:         ceClient,
		logger:           logger,
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

type adapter struct {
	awsArnString         string
	awsArn               arn.ARN
	awsDynamoDBTableName string
	config               *aws.Config
	session              *session.Session
	dynamoDB             *dynamodb.DynamoDB

	replier          *targetce.Replier
	discardCEContext bool
	ceClient         cloudevents.Client
	logger           *zap.SugaredLogger
}

func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting AWS DynamoDB Target adapter")
	s := session.Must(session.NewSession(a.config))
	a.session = s

	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

// Parse and send the aws event
func (a *adapter) dispatch(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var eventJSONMap map[string]interface{}

	if a.discardCEContext {
		if err := event.DataAs(&eventJSONMap); err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}
	} else {
		b, err := json.Marshal(event)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}
		if err := json.Unmarshal(b, &eventJSONMap); err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}
	}

	av, err := dynamodbattribute.MarshalMap(eventJSONMap)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, fmt.Errorf("Error marshalling attribute"), nil)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: &a.awsDynamoDBTableName,
	}

	resp, err := a.dynamoDB.PutItem(input)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	return a.replier.Ok(&event, resp)
}
