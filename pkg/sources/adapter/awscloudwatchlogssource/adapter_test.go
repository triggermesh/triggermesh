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

package awscloudwatchlogssource

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/stretchr/testify/assert"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"
)

const tLogGroupArnResource = "2020/12/12/[$LATEST]e70494fac3ba43c7b859fc722b061d33"

type mockedCloudWatchLogsClient struct {
	cloudwatchlogsiface.CloudWatchLogsAPI

	StreamsResp cloudwatchlogs.DescribeLogStreamsOutput
	EventsResp  cloudwatchlogs.GetLogEventsOutput
	err         error
}

func (m mockedCloudWatchLogsClient) DescribeLogStreamsPages(input *cloudwatchlogs.DescribeLogStreamsInput, fn func(*cloudwatchlogs.DescribeLogStreamsOutput, bool) bool) error {
	fn(&m.StreamsResp, true)

	return m.err
}

func (m mockedCloudWatchLogsClient) GetLogEventsPages(input *cloudwatchlogs.GetLogEventsInput, fn func(*cloudwatchlogs.GetLogEventsOutput, bool) bool) error {
	fn(&m.EventsResp, true)

	return m.err
}

func TestExtractLogDetails(t *testing.T) {
	testCases := []struct {
		ArnResource    string
		ExpectedGroup  string
		ExpectedStream string
	}{
		{
			ArnResource:    "log-group:/aws/lambda/lambdadumper:*",
			ExpectedGroup:  "/aws/lambda/lambdadumper",
			ExpectedStream: "",
		},
		{
			ArnResource:    "log-group:/aws/lambda/lambdadumper:log-stream:2020/12/12/[$LATEST]e70494fac3ba43c7b859fc722b061d33",
			ExpectedGroup:  "/aws/lambda/lambdadumper",
			ExpectedStream: "2020/12/12/[$LATEST]e70494fac3ba43c7b859fc722b061d33",
		},
		{
			ArnResource:    "log-group:/aws/lambda/lambdadumper",
			ExpectedGroup:  "/aws/lambda/lambdadumper",
			ExpectedStream: "",
		},
	}

	for _, tt := range testCases {
		group, stream := ExtractLogDetails(tt.ArnResource)

		assert.EqualValues(t, tt.ExpectedGroup, group)
		assert.EqualValues(t, tt.ExpectedStream, stream)
	}
}

func TestAdapterCollectLogsBaseCase(t *testing.T) {
	now := time.Now()
	ceClient := adaptertest.NewTestClient()
	duration, _ := time.ParseDuration("1m")

	a := &adapter{
		logger:          loggingtesting.TestLogger(t),
		ceClient:        ceClient,
		cwLogsClient:    mockedCloudWatchLogsClient{},
		pollingInterval: duration,
	}

	ctx := context.Background()

	a.CollectLogs(ctx, nil, now)
	events := ceClient.Sent()
	assert.Len(t, events, 0)
}

func TestAdapterCollectLogs(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-time.Minute).Unix() * 1000
	ceClient := adaptertest.NewTestClient()
	duration, _ := time.ParseDuration("2m")
	logStreamArn := makeARN(tLogGroupArnResource)
	logStreamName := "2020/12/15/[$LATEST]6b76c61acb68425f8e2f08156bc44e27"
	testString := "hello world"

	outputEvent := cloudwatchlogs.OutputLogEvent{
		IngestionTime: &startTime,
		Message:       &testString,
		Timestamp:     &startTime,
	}

	a := &adapter{
		logger: loggingtesting.TestLogger(t),

		ceClient: ceClient,
		cwLogsClient: mockedCloudWatchLogsClient{
			StreamsResp: cloudwatchlogs.DescribeLogStreamsOutput{
				LogStreams: []*cloudwatchlogs.LogStream{{
					Arn:                 aws.String(logStreamArn.String()),
					CreationTime:        nil,
					FirstEventTimestamp: &startTime,
					LastEventTimestamp:  &startTime,
					LastIngestionTime:   &startTime,
					LogStreamName:       &logStreamName,
					UploadSequenceToken: nil,
				}},
				NextToken: nil,
			},
			EventsResp: cloudwatchlogs.GetLogEventsOutput{
				Events:            []*cloudwatchlogs.OutputLogEvent{&outputEvent},
				NextBackwardToken: nil,
				NextForwardToken:  nil,
			},
		},

		arn: logStreamArn,

		pollingInterval: duration,
	}

	ctx := context.Background()

	a.CollectLogs(ctx, nil, now)
	events := ceClient.Sent()
	assert.Len(t, events, 1)

	assert.EqualValues(t, events[0].Type(), "com.amazon.logs.log")
	assert.EqualValues(t, events[0].Source(), logStreamArn.String())

	var logRecord cloudwatchlogs.OutputLogEvent
	err := events[0].DataAs(&logRecord)
	assert.NoError(t, err)
	assert.EqualValues(t, outputEvent, logRecord)
}

// makeARN returns a fake CloudWatch Log Group ARN for the given resource.
func makeARN(resource string) arn.ARN {
	return arn.ARN{
		Partition: "aws",
		Service:   "logs",
		Region:    "us-fake-0",
		AccountID: "123456789012",
		Resource:  resource,
	}
}
