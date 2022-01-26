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
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	loggingtesting "knative.dev/pkg/logging/testing"
)

const (
	tQueueArnResource = "MyQueue"

	expectedSucessfulEvent  = "Context Attributes,\n  specversion: 1.0\n  type: io.triggermesh.targets.aws.sqs.result\n  source: arn:aws:sqs:us-west-2:925906438773:DemoQueue\n  id: \n  datacontenttype: application/json\nData,\n  \"{\\n  MD5OfMessageBody: \\\"098f6bcd4621d373cade4e832627b4f6\\\",\\n  MessageId: \\\"00000000-0000-0000-0000-0000000001\\\",\\n  SequenceNumber: \\\"1\\\"\\n}\"\n"
	expectedFailureResponse = "500: error publishing to sqs"

	tMsgIDPrefix = "00000000-0000-0000-0000-000000000" // + 3 digits appended for each msg
)

func TestSuccessfulRequest(t *testing.T) {
	arn := makeARN(tQueueArnResource)
	testCases := map[string]struct {
		inEvent          cloudevents.Event
		failSendMessage  bool
		expectedResponse string
		expectedEvent    string
		client           *standardMockSQSClient
	}{
		"Successful request": {
			inEvent:          newEvent(t),
			failSendMessage:  false,
			expectedResponse: "",
			expectedEvent:    expectedSucessfulEvent,
			client:           &standardMockSQSClient{failSendMessage: false},
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			a := adapter{
				logger:           loggingtesting.TestLogger(t),
				sqsClient:        tc.client,
				awsArnString:     "arn:aws:sqs:us-west-2:925906438773:DemoQueue",
				awsArn:           arn,
				discardCEContext: false,
			}
			event, response := a.dispatch(tc.inEvent)
			if event != nil {
				assert.Equal(t, tc.expectedEvent, event.String())
			}
			assert.Equal(t, tc.expectedResponse, response.Error())
			assert.Lenf(t, tc.client.inputRecorder, 1, "Client records a single request")
		})
	}
}

func TestFailureRequest(t *testing.T) {
	arn := makeARN(tQueueArnResource)
	testCases := map[string]struct {
		inEvent          cloudevents.Event
		failSendMessage  bool
		expectedResponse string
		expectedEvent    string
		client           *standardMockSQSClient
	}{
		"Failure request": {
			inEvent:          newEvent(t),
			failSendMessage:  true,
			expectedResponse: expectedFailureResponse,
			expectedEvent:    "",
			client:           &standardMockSQSClient{failSendMessage: true},
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			a := adapter{
				logger:           loggingtesting.TestLogger(t),
				sqsClient:        tc.client,
				awsArnString:     "arn:aws:sqs:us-west-2:925906438773:DemoQueue",
				awsArn:           arn,
				discardCEContext: false,
			}
			event, response := a.dispatch(tc.inEvent)
			if event != nil {
				assert.Equal(t, tc.expectedEvent, event.String())
			}
			assert.Equal(t, tc.expectedResponse, response.Error())
		})
	}
}

func newEvent(t *testing.T) cloudevents.Event {
	t.Helper()
	ce := cloudevents.NewEvent()
	ce.SetID("1234567890")
	ce.SetSource("test.source")
	ce.SetType("test.type")
	if err := ce.SetData(cloudevents.TextPlain, "Lorem Ipsum"); err != nil {
		t.Fatalf("Failed to set event data: %s", err)
	}

	return ce
}

// makeARN returns a fake SQS ARN for the given resource.
func makeARN(resource string) arn.ARN {
	return arn.ARN{
		Partition: "aws",
		Service:   "sqs",
		Region:    "us-fake-0",
		AccountID: "123456789012",
		Resource:  resource,
	}
}

// standardMockSQSClient is a mocked SQS client which returns a standard set of
// responses and never errors.
type standardMockSQSClient struct {
	sqsiface.SQSAPI
	failSendMessage bool
	inputRecorder   []*sqs.SendMessageInput
}

func (c *standardMockSQSClient) SendMessage(sqsmsg *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	if c.failSendMessage {
		return &sqs.SendMessageOutput{}, assert.AnError
	}

	c.logEvent(sqsmsg)

	return &sqs.SendMessageOutput{
		MessageId:        aws.String(tMsgIDPrefix + "1"),
		SequenceNumber:   aws.String("1"),
		MD5OfMessageBody: aws.String("098f6bcd4621d373cade4e832627b4f6"),
	}, nil
}

func (c *standardMockSQSClient) logEvent(in *sqs.SendMessageInput) error {
	c.inputRecorder = append(c.inputRecorder, in)
	return nil
}
