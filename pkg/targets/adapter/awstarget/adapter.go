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

package awstarget

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

// Adapter implementation
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

	return &awsAdapter{
		config:               config, // define configuration for the aws client
		awsType:              env.AwsTargetType,
		awsArnString:         env.AwsTargetArn,
		awsArn:               a,
		awsDynamoDBTableName: dynamodbTable,
		awsKinesisPartition:  env.AwsKinesisPartition,
		discardCEContext:     env.DiscardCEContext,
		ceClient:             ceClient,

		logger: logger,
	}
}

var _ pkgadapter.Adapter = (*awsAdapter)(nil)

type awsAdapter struct {
	awsType              string
	awsArnString         string
	awsArn               arn.ARN
	awsDynamoDBTableName string
	awsKinesisPartition  string
	config               *aws.Config
	session              *session.Session
	lda                  *lambda.Lambda
	s3                   *s3.S3
	sns                  *sns.SNS
	sqs                  *sqs.SQS
	kinesis              *kinesis.Kinesis
	dynamoDB             *dynamodb.DynamoDB

	discardCEContext bool

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

func (a *awsAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting AWS adapter")
	s := session.Must(session.NewSession(a.config))
	a.session = s

	switch a.awsType {
	case "dynamodb":
		a.dynamoDB = dynamodb.New(s)
	case "lambda":
		a.lda = lambda.New(s)
	case "sns":
		a.sns = sns.New(s)
	case "sqs":
		a.sqs = sqs.New(s)
	case "kinesis":
		a.kinesis = kinesis.New(s)
	case "s3":
		a.s3 = s3.New(s)
	default:
		return errors.New("unknown aws service type: " + a.awsType)
	}

	if err := a.ceClient.StartReceiver(ctx, a.dispatch); err != nil {
		return err
	}
	return nil
}

// Parse and send the aws event
func (a *awsAdapter) dispatch(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var e *cloudevents.Event
	var r cloudevents.Result

	switch a.awsType {
	case "dynamodb":
		e, r = a.dispatchDynamoDB(event)
	case "lambda":
		e, r = a.dispatchLambda(event)
	case "sns":
		e, r = a.dispatchSNS(event)
	case "sqs":
		e, r = a.dispatchSQS(event)
	case "kinesis":
		e, r = a.dispatchKinesis(event)
	case "s3":
		e, r = a.dispatchS3(event)
	default:
		return a.reportError("unknown aws service type", nil)
	}

	return e, r
}

func (a *awsAdapter) dispatchDynamoDB(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
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

	resp, err := a.dynamoDB.PutItem(input)
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

func (a *awsAdapter) dispatchLambda(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
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

func (a *awsAdapter) dispatchSNS(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
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

	result, err := a.sns.Publish(&sns.PublishInput{
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

func (a *awsAdapter) dispatchSQS(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
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

	result, err := a.sqs.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(msg)),
		QueueUrl:    &url,
	})

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

func (a *awsAdapter) dispatchKinesis(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var data []byte

	if a.discardCEContext {
		data = event.Data()
	} else {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			return a.reportError("Error marshalling CloudEvent", err)
		}
		data = jsonEvent
	}

	// Stream name must be present, however the ARN encodes the resource as stream/<stream_name>
	streamName := strings.Split(a.awsArn.Resource, "/")
	if len(streamName) != 2 {
		return a.reportError("unable to extract kinesis stream name from ARN", nil)
	}

	result, err := a.kinesis.PutRecord(&kinesis.PutRecordInput{
		Data:         data,
		PartitionKey: &a.awsKinesisPartition,
		StreamName:   &streamName[1],
	})

	if err != nil {
		return a.reportError("error publishing to kinesis", err)
	}

	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)
	err = responseEvent.SetData(cloudevents.ApplicationJSON, result.GoString())
	if err != nil {
		return a.reportError("error generating response event", err)
	}

	responseEvent.SetType("io.triggermesh.targets.aws.kinesis.result")
	responseEvent.SetSource(a.awsArnString)

	return &responseEvent, cloudevents.ResultACK
}

// Treat the event as an object to upload. The bucket will be defined as a part
// of the ARN. The `subject` attribute of the received CloudEvent is required
// to indicate what bucket key should be used.
//
// When the `type` attribute of the received CloudEvent is `io.triggermesh.awss3.object.put`,
// only the CloudEvent data (without context attributes) is stored in the
// destination S3 object, regardless of the value of the `discardCloudEventContext`
// spec attribute.
//
func (a *awsAdapter) dispatchS3(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var dataReader *bytes.Reader

	if event.Type() == v1alpha1.EventTypeAWSS3Put || a.discardCEContext {
		dataReader = bytes.NewReader(event.Data())
	} else {
		d, err := json.Marshal(event)
		if err != nil {
			return a.reportError("error marshalling CloudEvent", err)
		}
		dataReader = bytes.NewReader(d)
	}

	key := event.Subject()
	if key == "" {
		key = event.Type() + "/" + event.Source() + "/" + event.Time().String()
	}

	bucket := strings.Split(a.awsArn.Resource, "/")[0]

	putInput := s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   dataReader,
	}

	result, err := a.s3.PutObject(&putInput)

	if err != nil {
		return a.reportError("error publishing object to s3 bucket", err)
	}

	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)
	err = responseEvent.SetData(cloudevents.ApplicationJSON, result.GoString())
	if err != nil {
		return a.reportError("error generating response event", err)
	}

	responseEvent.SetType(v1alpha1.EventTypeAWSS3Result)
	responseEvent.SetSource(a.awsArnString)

	return &responseEvent, cloudevents.ResultACK
}

func (a *awsAdapter) reportError(msg string, err error) (*cloudevents.Event, cloudevents.Result) {
	a.logger.Errorw(msg, zap.Error(err))
	return nil, cloudevents.NewHTTPResult(http.StatusInternalServerError, msg)
}
