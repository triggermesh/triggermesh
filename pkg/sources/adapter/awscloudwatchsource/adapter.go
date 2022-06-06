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
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/health"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	Region string `envconfig:"AWS_REGION"`

	Query           string `envconfig:"QUERIES" required:"true"`          // JSON based array of name/query pairs
	PollingInterval string `envconfig:"POLLING_INTERVAL" required:"true"` // free tier is 5m

	// The environment variables below aren't read from the envConfig struct
	// by the AWS SDK, but rather directly using os.Getenv().
	// They are nevertheless listed here for documentation purposes.
	_ string `envconfig:"AWS_ACCESS_KEY_ID"`
	_ string `envconfig:"AWS_SECRET_ACCESS_KEY"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger *zap.SugaredLogger
	mt     *pkgadapter.MetricTag

	eventsource string

	cwClient cloudwatchiface.CloudWatchAPI
	ceClient cloudevents.Client

	metricQueries   []*cloudwatch.MetricDataQuery
	pollingInterval time.Duration
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.AWSCloudWatchSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	eventsource := v1alpha1.AWSCloudWatchSourceName(envAcc.GetNamespace(), envAcc.GetName())

	env := envAcc.(*envConfig)

	cfg := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(env.Region),
	))

	interval, err := time.ParseDuration(env.PollingInterval)
	if err != nil {
		logger.Panicf("Unable to parse interval duration: %v", zap.Error(err))
	}

	metricQueries, err := parseQueries(env.Query)
	if err != nil {
		logger.Panicf("unable to parse metric queries: %v", zap.Error(err))
	}

	return &adapter{
		logger:      logger,
		mt:          mt,
		eventsource: eventsource,

		cwClient: cloudwatch.New(cfg),
		ceClient: ceClient,

		pollingInterval: interval,
		metricQueries:   metricQueries,
	}
}

// parseQueries - Take the JSON representation of the query as passed in, and
// convert it into something useful to aws
func parseQueries(rawQuery string) ([]*cloudwatch.MetricDataQuery, error) {
	queries := make([]*cloudwatch.MetricDataQuery, 0)
	rawQueries := make([]v1alpha1.AWSCloudWatchMetricQuery, 0)

	err := json.Unmarshal([]byte(rawQuery), &rawQueries)
	if err != nil {
		return nil, err
	}

	for _, v := range rawQueries {
		name := v.Name

		if v.Expression != nil {
			queries = append(queries, &cloudwatch.MetricDataQuery{
				Expression: v.Expression,
				Id:         &name,
			})
		} else if v.Metric != nil {
			queries = append(queries, &cloudwatch.MetricDataQuery{
				Id:         &name,
				MetricStat: transformQuery(v.Metric),
			})
		}
	}
	return queries, nil
}

func transformQuery(q *v1alpha1.AWSCloudWatchMetricStat) *cloudwatch.MetricStat {
	dimensions := make([]*cloudwatch.Dimension, 0)

	for _, v := range q.Metric.Dimensions {
		dimensions = append(dimensions, &cloudwatch.Dimension{
			Name:  &v.Name,
			Value: &v.Value,
		})
	}

	ms := cloudwatch.MetricStat{
		Metric: &cloudwatch.Metric{
			MetricName: &q.Metric.MetricName,
			Namespace:  &q.Metric.Namespace,
			Dimensions: dimensions,
		},
		Period: &q.Period,
		Stat:   &q.Stat,
	}

	if q.Unit != "" {
		ms.SetUnit(q.Unit)
	}

	return &ms
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	if err := peekMetrics(ctx, a.cwClient); err != nil {
		return fmt.Errorf("unable to read metrics: %w", err)
	}

	health.MarkReady()

	a.logger.Info("Starting CloudWatch adapter")

	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	// Setup polling to retrieve metrics
	poll := time.NewTicker(a.pollingInterval)
	defer poll.Stop()

	// Wake up every pollingInterval, and retrieve the logs
	var priorTime *time.Time
	for {
		select {
		case <-ctx.Done():
			return nil

		case t := <-poll.C:
			go a.CollectMetrics(ctx, priorTime, t)
			priorTime = &t
		}
	}
}

func (a *adapter) CollectMetrics(ctx context.Context, priorTime *time.Time, currentTime time.Time) {
	a.logger.Debug("Firing metrics")
	startInterval := currentTime.Add(-a.pollingInterval)

	if priorTime != nil {
		startInterval = *priorTime
	}

	metricInput := cloudwatch.GetMetricDataInput{
		EndTime:           &currentTime,
		StartTime:         &startInterval,
		MetricDataQueries: a.metricQueries,
	}

	err := a.cwClient.GetMetricDataPages(&metricInput, func(output *cloudwatch.GetMetricDataOutput, b bool) bool {
		err := a.SendMetricEvent(ctx, output)
		if err != nil {
			a.logger.Errorw("Error sending metrics", zap.Error(err))
			return false
		}

		// Ensure that we indicate if there's more work to do
		return !b
	})
	if err != nil {
		a.logger.Errorw("Error retrieving metrics", zap.Error(err))
		return
	}
}

func (a *adapter) SendMetricEvent(ctx context.Context, metricOutput *cloudwatch.GetMetricDataOutput) error {
	// multiple messages or messages and metric data, and insure the CloudEvent
	// ID is common.

	for _, v := range metricOutput.Messages {
		event := cloudevents.NewEvent(cloudevents.VersionV1)
		event.SetType(v1alpha1.AWSEventType(v1alpha1.ServiceCloudWatch, v1alpha1.AWSCloudWatchMessageEventType))
		event.SetSource(a.eventsource)
		err := event.SetData(cloudevents.ApplicationJSON, v)

		if err != nil {
			return fmt.Errorf("failed to set event data: %w", err)
		}

		if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
			return result
		}
	}

	for _, v := range metricOutput.MetricDataResults {
		event := cloudevents.NewEvent(cloudevents.VersionV1)
		event.SetType(v1alpha1.AWSEventType(v1alpha1.ServiceCloudWatch, v1alpha1.AWSCloudWatchMetricEventType))
		event.SetSource(a.eventsource)
		err := event.SetData(cloudevents.ApplicationJSON, v)

		if err != nil {
			return fmt.Errorf("failed to set event data: %w", err)
		}

		if result := a.ceClient.Send(context.Background(), event); !cloudevents.IsACK(result) {
			return result
		}
	}

	return nil
}

// peekMetrics verifies that the provided client can read metrics from CloudWatch.
func peekMetrics(ctx context.Context, cli cloudwatchiface.CloudWatchAPI) error {
	const oneHourInSeconds = 3600

	_, err := cli.GetMetricDataWithContext(ctx, &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(time.Unix(0, 0)),
		EndTime:   aws.Time(time.Unix(1, 0)),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			// This query is technically valid but we don't need it
			// to return any result.
			{
				Id: aws.String("peek"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						MetricName: aws.String("Peak"),
						Namespace:  aws.String("TriggerMesh"),
					},
					Period: aws.Int64(oneHourInSeconds),
					Stat:   aws.String(cloudwatch.StatisticSum),
				},
			},
		},
	})
	return err
}
