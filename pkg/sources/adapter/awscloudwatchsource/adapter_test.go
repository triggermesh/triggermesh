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

package awscloudwatchsource

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/stretchr/testify/assert"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

const (
	tNs   = "test-namespace"
	tName = "test-source"
)

type mockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI

	Resp cloudwatch.GetMetricDataOutput
	err  error
}

func (m mockCloudWatchClient) GetMetricDataPages(input *cloudwatch.GetMetricDataInput, fn func(*cloudwatch.GetMetricDataOutput, bool) bool) error {
	fn(&m.Resp, true)

	return m.err
}

// TestParseQueries Given a query string, ensure that
func TestParseQueries(t *testing.T) {
	const (
		queryStr = "[{\"name\":\"testquery\",\"metric\":{\"period\":60,\"stat\":\"Sum\",\"metric\":{\"dimensions\":[{\"name\":\"FunctionName\",\"value\":\"makemoney\"}],\"metricName\":\"Duration\",\"namespace\":\"AWS/Lambda\"}}}]"

		name            = "testquery"
		period          = int64(60)
		stat            = "Sum"
		metricName      = "Duration"
		metricNamespace = "AWS/Lambda"
		dimensionName   = "FunctionName"
		dimensionValue  = "makemoney"
	)

	metricQuery := cloudwatch.MetricDataQuery{
		Expression: nil,
		Id:         aws.String(name),
		MetricStat: &cloudwatch.MetricStat{
			Metric: &cloudwatch.Metric{
				Dimensions: []*cloudwatch.Dimension{{
					Name:  aws.String(dimensionName),
					Value: aws.String(dimensionValue),
				}},
				MetricName: aws.String(metricName),
				Namespace:  aws.String(metricNamespace),
			},
			Period: aws.Int64(period),
			Stat:   aws.String(stat),
		},
	}

	results, err := parseQueries(queryStr)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	m := results[0]
	assert.EqualValues(t, metricQuery, *m)
}

func TestCollectMetrics(t *testing.T) {
	ceClient := adaptertest.NewTestClient()

	const (
		metricID         = "testmetrics"
		metricNamespace  = "AWS/Lambda"
		metricLabel      = "Duration"
		metricStatusCode = "Complete"
		val              = float64(37.566818845509246)
	)

	ts := time.Date(1970, 1, 1, 12, 0, 0, 0, time.UTC)

	const (
		dimensionName  = "FunctionName"
		dimensionValue = "makemoney"
		period         = int64(60)
		stat           = "Sum"
	)

	const pollingInterval = time.Minute

	a := &adapter{
		logger:      loggingtesting.TestLogger(t),
		eventsource: v1alpha1.AWSCloudWatchSourceName(tNs, tName),

		ceClient: ceClient,
		metricQueries: []*cloudwatch.MetricDataQuery{{
			Expression: nil,
			Id:         aws.String(metricID),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Dimensions: []*cloudwatch.Dimension{{
						Name:  aws.String(dimensionName),
						Value: aws.String(dimensionValue),
					}},
					MetricName: aws.String(metricLabel),
					Namespace:  aws.String(metricNamespace),
				},
				Period: aws.Int64(period),
				Stat:   aws.String(stat),
			},
		}},
		cwClient: mockCloudWatchClient{
			Resp: cloudwatch.GetMetricDataOutput{
				Messages: nil,
				MetricDataResults: []*cloudwatch.MetricDataResult{{
					Id:         aws.String(metricID),
					Label:      aws.String(metricLabel),
					Messages:   nil,
					StatusCode: aws.String(metricStatusCode),
					Timestamps: []*time.Time{&ts},
					Values:     []*float64{aws.Float64(val)},
				}},
				NextToken: nil,
			},
			err: nil,
		},

		pollingInterval: pollingInterval,
	}

	metricOutput := cloudwatch.GetMetricDataOutput{
		Messages: nil,
		MetricDataResults: []*cloudwatch.MetricDataResult{{
			Id:         aws.String(metricID),
			Label:      aws.String(metricLabel),
			Messages:   nil,
			StatusCode: aws.String(metricStatusCode),
			Timestamps: []*time.Time{&ts},
			Values:     []*float64{aws.Float64(val)},
		}},
		NextToken: nil,
	}

	ctx := context.Background()

	a.CollectMetrics(ctx, nil, time.Now())

	events := ceClient.Sent()
	assert.Len(t, events, 1)

	assert.EqualValues(t, events[0].Type(), "com.amazon.cloudwatch.metrics.metric")
	assert.EqualValues(t, events[0].Source(), "io.triggermesh.awscloudwatchsource.test-namespace.test-source")

	var metricRecord cloudwatch.MetricDataResult
	err := events[0].DataAs(&metricRecord)

	assert.NoError(t, err)
	assert.EqualValues(t, *metricOutput.MetricDataResults[0], metricRecord)
}

func TestSendMetricEvent(t *testing.T) {
	ceClient := adaptertest.NewTestClient()

	a := &adapter{
		logger:      loggingtesting.TestLogger(t),
		eventsource: v1alpha1.AWSCloudWatchSourceName(tNs, tName),
		ceClient:    ceClient,
	}

	const (
		metricID         = "testmetrics"
		metricLabel      = "Duration"
		metricStatusCode = "Complete"
		val              = float64(37.566818845509246) // must keep this cast to ensure proper [de]serialization
	)

	ts := time.Date(1970, 1, 1, 12, 0, 0, 0, time.UTC)

	metricOutput := cloudwatch.GetMetricDataOutput{
		Messages: nil,
		MetricDataResults: []*cloudwatch.MetricDataResult{{
			Id:         aws.String(metricID),
			Label:      aws.String(metricLabel),
			Messages:   nil,
			StatusCode: aws.String(metricStatusCode),
			Timestamps: []*time.Time{&ts},
			Values:     []*float64{aws.Float64(val)},
		}},
		NextToken: nil,
	}

	ctx := context.Background()

	err := a.SendMetricEvent(ctx, &metricOutput)
	assert.NoError(t, err)
	events := ceClient.Sent()
	assert.Len(t, events, 1)

	assert.EqualValues(t, events[0].Type(), "com.amazon.cloudwatch.metrics.metric")
	assert.EqualValues(t, events[0].Source(), "io.triggermesh.awscloudwatchsource.test-namespace.test-source")

	var metricRecord cloudwatch.MetricDataResult
	err = events[0].DataAs(&metricRecord)

	assert.NoError(t, err)
	assert.EqualValues(t, *metricOutput.MetricDataResults[0], metricRecord)
}

func TestSendMessageEvent(t *testing.T) {
	ceClient := adaptertest.NewTestClient()

	a := &adapter{
		logger:      loggingtesting.TestLogger(t),
		eventsource: v1alpha1.AWSCloudWatchSourceName(tNs, tName),
		ceClient:    ceClient,
	}

	msgCode := "Success"
	msgValue := "This is a sample message value"

	metricOutput := cloudwatch.GetMetricDataOutput{
		Messages: []*cloudwatch.MessageData{{
			Code:  &msgCode,
			Value: &msgValue,
		}},
		NextToken: nil,
	}

	ctx := context.Background()

	err := a.SendMetricEvent(ctx, &metricOutput)
	assert.NoError(t, err)
	events := ceClient.Sent()
	assert.Len(t, events, 1)

	assert.EqualValues(t, events[0].Type(), "com.amazon.cloudwatch.metrics.message")
	assert.EqualValues(t, events[0].Source(), "io.triggermesh.awscloudwatchsource.test-namespace.test-source")

	var metricRecord cloudwatch.MessageData
	err = events[0].DataAs(&metricRecord)

	assert.NoError(t, err)
	assert.EqualValues(t, *metricOutput.Messages[0], metricRecord)
}
