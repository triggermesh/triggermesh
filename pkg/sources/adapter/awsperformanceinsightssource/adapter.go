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

package awsperformanceinsightssource

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pi"
	"github.com/aws/aws-sdk-go/service/pi/piiface"
	"github.com/aws/aws-sdk-go/service/rds"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/health"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	ARN string `envconfig:"ARN"`

	PollingInterval string `envconfig:"POLLING_INTERVAL" required:"true"`

	Metrics []string `envconfig:"PI_METRICS" required:"true"`

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

	pIClient *pi.PI
	ceClient cloudevents.Client

	arn             arn.ARN
	pollingInterval time.Duration
	metricQueries   []*pi.MetricQuery
	resourceID      string
}

// event represents the structured event data to be sent as the payload of the Cloudevent
type event struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.AWSPerformanceInsightsSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	a := common.MustParseARN(env.ARN)

	cfg := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(a.Region),
	))

	interval, err := time.ParseDuration(env.PollingInterval)
	if err != nil {
		logger.Panicf("Unable to parse interval duration: %v", err)
	}

	var mql []*pi.MetricQuery

	for _, m := range env.Metrics {
		mq := &pi.MetricQuery{Metric: aws.String(m)}
		mql = append(mql, mq)
	}

	r := rds.New(cfg)

	dbi, err := r.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		Filters: []*rds.Filter{
			{
				Name:   aws.String("db-instance-id"),
				Values: aws.StringSlice([]string{a.String()}),
			},
		},
	})
	if err != nil {
		logger.Panicf("Unable to describe DB instances: %v", err)
	}

	if len(dbi.DBInstances) != 1 {
		logger.Panicf("DB instance %s not found", a.String())
	}

	resourceID := *dbi.DBInstances[0].DbiResourceId

	return &adapter{
		logger: logger,
		mt:     mt,

		pIClient: pi.New(cfg),
		ceClient: ceClient,

		arn: a,

		pollingInterval: interval,
		metricQueries:   mql,
		resourceID:      resourceID,
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	if err := peekResourceMetrics(ctx, a.pIClient, a.resourceID); err != nil {
		return fmt.Errorf("unable to read resource metrics: %w", err)
	}

	health.MarkReady()

	a.logger.Info("Enabling AWS Performance Insights Source")

	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	// Setup polling to retrieve metrics
	poll := time.NewTicker(a.pollingInterval)
	defer poll.Stop()

	// Wake up every pollingInterval, and retrieve the logs
	var priorTime time.Time
	priorTime = time.Now()
	for {
		select {
		case <-ctx.Done():
			return nil

		case t := <-poll.C:
			go a.PollMetrics(ctx, priorTime, t)
			priorTime = t
		}
	}
}

func (a *adapter) PollMetrics(ctx context.Context, priorTime time.Time, currentTime time.Time) {
	rmi := &pi.GetResourceMetricsInput{
		EndTime:       aws.Time(time.Now()),
		StartTime:     aws.Time(priorTime),
		Identifier:    aws.String(a.resourceID),
		MetricQueries: a.metricQueries,
		ServiceType:   aws.String(pi.ServiceTypeRds),
	}

	rm, err := a.pIClient.GetResourceMetrics(rmi)

	if err != nil {
		a.logger.Errorw("Error retrieving resource metrics", zap.Error(err))
		return
	}

	for _, d := range rm.MetricList {
		for _, metric := range d.DataPoints {
			if metric.Value != nil {
				e := &event{
					Metric: *d.Key.Metric,
					Value:  *metric.Value,
				}

				event := cloudevents.NewEvent(cloudevents.VersionV1)
				event.SetType(v1alpha1.AWSPerformanceInsightsGenericEventType)
				event.SetSource(a.arn.String())
				event.SetExtension("pimetric", d.Key.Metric)
				if err := event.SetData(cloudevents.ApplicationJSON, e); err != nil {
					a.logger.Errorw("Failed to set event data", zap.Error(err))
					return
				}

				if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
					a.logger.Errorw("Failed to send event data", zap.Error(err))
					return
				}

				a.logger.Debug("Sent Cloudevent Successfully")
			}
		}
	}
}

// peekResourceMetrics verifies that there are metrics available for the given
// DB instance.
func peekResourceMetrics(ctx context.Context, cli piiface.PIAPI, dbInstanceID string) error {
	_, err := cli.ListAvailableResourceMetricsWithContext(ctx, &pi.ListAvailableResourceMetricsInput{
		Identifier:  &dbInstanceID,
		MetricTypes: aws.StringSlice([]string{"os", "db"}),
		ServiceType: aws.String(pi.ServiceTypeRds),
		MaxResults:  aws.Int64(1),
	})
	return err
}
