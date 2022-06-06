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
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"

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

	cwLogsClient cloudwatchlogsiface.CloudWatchLogsAPI
	ceClient     cloudevents.Client

	arn arn.ARN

	pollingInterval time.Duration
	logGroup        string
	logStream       string
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.AWSCloudWatchLogsSourceResource.String(),
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
		logger.Panicf("Unable to parse interval duration: %v", zap.Error(err))
	}

	logGroup, logStream := ExtractLogDetails(a.Resource)

	return &adapter{
		logger: logger,
		mt:     mt,

		cwLogsClient: cloudwatchlogs.New(cfg),
		ceClient:     ceClient,

		arn: a,

		pollingInterval: interval,
		logGroup:        logGroup,
		logStream:       logStream,
	}
}

// ExtractLogDetails takes the resource string from the ARN, and extract the `log-group` and `log-stream`.
func ExtractLogDetails(details string) (string, string) {
	atoms := strings.Split(details, ":")

	var logGroup string
	var logStream string

	for i, k := range atoms {
		switch k {
		case "log-group":
			logGroup = atoms[i+1]
		case "log-stream":
			logStream = atoms[i+1]
		}
	}

	return logGroup, logStream
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	if err := peekLogGroup(ctx, a.cwLogsClient, a.logGroup); err != nil {
		return fmt.Errorf("unable to access log group %q: %w", a.logGroup, err)
	}

	health.MarkReady()

	a.logger.Info("Starting CloudWatch Log adapter")

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
			go a.CollectLogs(ctx, priorTime, t)
			priorTime = &t
		}
	}
}

// CollectLogs receives events from CloudWatch client and sends them to a sink.
func (a *adapter) CollectLogs(ctx context.Context, priorTime *time.Time, currentTime time.Time) {
	a.logger.Debug("Firing logs")
	startTime := currentTime.Add(-a.pollingInterval).Unix() * 1000

	if priorTime != nil {
		startTime = (*priorTime).Unix() * 1000
	}

	endTime := currentTime.Unix() * 1000

	logStreams := cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: &a.logGroup,
	}

	err := a.cwLogsClient.DescribeLogStreamsPages(&logStreams, func(output *cloudwatchlogs.DescribeLogStreamsOutput, b bool) bool {
		var logRequest *cloudwatchlogs.GetLogEventsInput

		for _, v := range output.LogStreams {
			if v.LastIngestionTime != nil && *v.LastIngestionTime > startTime {

				if a.logStream != "" && a.logStream != "*" && *v.LogStreamName != a.logStream {
					continue
				}
				logRequest = &cloudwatchlogs.GetLogEventsInput{
					EndTime:       &endTime,
					LogGroupName:  &a.logGroup,
					LogStreamName: v.LogStreamName,
					StartTime:     &startTime,
				}
			} else {
				continue
			}

			// Send out an event for every entry
			err := a.cwLogsClient.GetLogEventsPages(logRequest, func(logOutput *cloudwatchlogs.GetLogEventsOutput, lastPage bool) bool {
				// If there are no entries, then skip sending events
				if len(logOutput.Events) == 0 {
					a.logger.Debug("no log events sent")
					return !lastPage
				}

				// Ensure the entries captured within our range are the only events being published
				trimmedLogOutput := make([]*cloudwatchlogs.OutputLogEvent, 0)
				for _, v := range logOutput.Events {
					if *v.Timestamp >= startTime && *v.Timestamp < endTime {
						trimmedLogOutput = append(trimmedLogOutput, v)
					}
				}

				for _, v := range trimmedLogOutput {
					event := cloudevents.NewEvent(cloudevents.VersionV1)
					event.SetType(v1alpha1.AWSEventType(a.arn.Service, v1alpha1.AWSCloudWatchLogsGenericEventType))
					event.SetSource(a.arn.String())

					err := event.SetData(cloudevents.ApplicationJSON, v)
					if err != nil {
						a.logger.Errorw("Failed to set event data", zap.Error(err))
						return false
					}

					if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
						a.logger.Errorw("Failed to send event", zap.Error(err))
						return false
					}
				}

				return !lastPage
			})

			if err != nil {
				a.logger.Errorw("Error retrieving logs", zap.Error(err))
			}
		}

		return !b
	})

	if err != nil {
		a.logger.Errorw("Error retrieving log streams", zap.Error(err))
	}
}

// peekLogGroup verifies that a log group exists.
func peekLogGroup(ctx context.Context, cli cloudwatchlogsiface.CloudWatchLogsAPI, logGroup string) error {
	_, err := cli.DescribeLogStreamsWithContext(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: &logGroup,
		Limit:        aws.Int64(1),
	})
	return err
}
