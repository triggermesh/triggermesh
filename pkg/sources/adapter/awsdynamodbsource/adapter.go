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

package awsdynamodbsource

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams/dynamodbstreamsiface"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/health"
)

const (
	streamRecheckPeriod = 15 * time.Second
	getRecordsPeriod    = 3 * time.Second
)

// CloudEvents extensions
const (
	// The type of data modification that was performed on the DynamoDB table
	// https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_streams_Record.html#DDB-Type-streams_Record-eventName
	ceExtDynamoDBOperation = "dynamodboperation"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	ARN string `envconfig:"ARN" required:"true"`

	// Assume this IAM Role when access keys provided.
	AssumeIamRole string `envconfig:"AWS_ASSUME_ROLE_ARN"`

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

	dyndbClient    dynamodbiface.DynamoDBAPI
	dyndbStrClient dynamodbstreamsiface.DynamoDBStreamsAPI
	ceClient       cloudevents.Client

	arn arn.ARN

	// tracker for running records processors
	processors sync.Map
	wg         sync.WaitGroup

	lastStreamARN    *string
	lastStreamStatus *string
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.AWSDynamoDBSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	arn := common.MustParseARN(env.ARN)

	sess := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(arn.Region),
	))

	config := &aws.Config{}
	if env.AssumeIamRole != "" {
		config.Credentials = stscreds.NewCredentials(sess, env.AssumeIamRole)
	}

	return &adapter{
		logger: logger,
		mt:     mt,

		dyndbClient:    dynamodb.New(sess, config),
		dyndbStrClient: dynamodbstreams.New(sess, config),
		ceClient:       ceClient,

		arn: arn,
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	if _, err := a.getLatestStreamARN(ctx); err != nil {
		return fmt.Errorf("verifying stream for table %q: %w", a.arn, err)
	}

	health.MarkReady()

	a.logger.Info("Starting collection of DynamoDB records for table ", a.arn)

	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	t := time.NewTimer(0)
	defer t.Stop()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop

		case <-t.C:
			streamARN, err := a.getLatestStreamARN(ctx)
			if err != nil {
				return fmt.Errorf("retrieving stream ARN for table %s: %w", a.arn, err)
			}

			if a.lastStreamARN != nil && *streamARN != *a.lastStreamARN {
				a.logger.Warn("Active stream changed from ", *a.lastStreamARN, " to ", *streamARN)
				a.lastStreamStatus = nil
			}
			a.lastStreamARN = streamARN

			if err := a.recheckStream(ctx, streamARN); err != nil {
				a.logger.Errorw("Error while re-checking stream "+*streamARN, zap.Error(err))
			}

			t.Reset(streamRecheckPeriod)
		}
	}

	a.logger.Debug("Waiting for termination of records processors")
	a.wg.Wait()

	return nil
}

// errNoStream is an error type returned when a DynamoDB table doesn't have a
// stream associated with it.
type errNoStream /*table ARN*/ arn.ARN

// Error implements the error interface.
func (e errNoStream) Error() string {
	return fmt.Sprint("no stream is associated with table ", arn.ARN(e))
}

// getLatestStreamARN returns the ARN of the latest stream for the table.
func (a *adapter) getLatestStreamARN(ctx context.Context) (*string, error) {
	table, err := a.dyndbClient.DescribeTableWithContext(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(common.MustParseDynamoDBResource(a.arn.Resource)),
	})
	if err != nil {
		return nil, fmt.Errorf("retrieving table info: %w", err)
	}

	if table.Table.LatestStreamArn == nil {
		return nil, errNoStream(a.arn)
	}

	return table.Table.LatestStreamArn, nil
}

// recheckStream ensures a records processor is running for each of the stream's shards.
func (a *adapter) recheckStream(ctx context.Context, streamARN *string) error {
	a.logger.Debug("Checking stream for new shards")

	var lastEvaluatedShardID *string

	for {
		stream, err := a.dyndbStrClient.DescribeStreamWithContext(ctx, &dynamodbstreams.DescribeStreamInput{
			StreamArn:             streamARN,
			ExclusiveStartShardId: lastEvaluatedShardID,
		})
		if err != nil {
			return fmt.Errorf("describing stream: %w", err)
		}

		streamStatus := stream.StreamDescription.StreamStatus

		switch {
		case a.lastStreamStatus != nil && *streamStatus != *a.lastStreamStatus:
			a.logger.Warn("Stream status changed from ", *a.lastStreamStatus, " to ", *streamStatus)

		case a.lastStreamStatus == nil && *streamStatus != dynamodbstreams.StreamStatusEnabled:
			a.logger.Warn("Stream has status ", *streamStatus, ". No records collection will occur")
		}

		a.lastStreamStatus = streamStatus

		if *streamStatus != dynamodbstreams.StreamStatusEnabled {
			return nil
		}

		for _, s := range stream.StreamDescription.Shards {
			a.ensureRecordsProcessor(ctx, streamARN, s.ShardId)
		}

		lastEvaluatedShardID = stream.StreamDescription.LastEvaluatedShardId

		// If LastEvaluatedShardId is nil, then the "last page" of results has been
		// processed and there is currently no more data to be retrieved.
		if lastEvaluatedShardID == nil {
			break
		}
	}

	return nil
}

// ensureRecordsProcessor ensures a records processor is running for the given shard.
func (a *adapter) ensureRecordsProcessor(ctx context.Context, streamARN *string, shardID *string) {
	if _, running := a.processors.LoadOrStore(*shardID, struct{}{}); running {
		a.logger.Debug("Record processor already running for shard ID ", *shardID)
		return
	}

	a.wg.Add(1)

	go func() {
		defer a.processors.Delete(*shardID)
		defer a.wg.Done()

		a.logger.Debug("Starting records processor for shard ID ", *shardID)

		if err := a.runRecordsProcessor(ctx, streamARN, shardID); err != nil {
			a.logger.Errorw("Records processor for shard ID "+*shardID+" returned with error", zap.Error(err))
			return
		}

		a.logger.Debug("Records processor for shard ID " + *shardID + " has stopped")
	}()
}

// runRecordsProcessor runs a records processor for the given shard.
func (a *adapter) runRecordsProcessor(ctx context.Context, streamARN *string, shardID *string) error {
	si, err := a.dyndbStrClient.GetShardIteratorWithContext(ctx, &dynamodbstreams.GetShardIteratorInput{
		StreamArn:         streamARN,
		ShardId:           shardID,
		ShardIteratorType: aws.String(dynamodbstreams.ShardIteratorTypeLatest),
	})
	if err != nil {
		return fmt.Errorf("getting shard iterator for shard ID %s: %w", *shardID, err)
	}

	// Use a timer to pace GetRecords API calls after all observed records
	// have been processed.
	// Each GetRecords API call is billed as a "streams read request" unit.
	// The free tier includes 2,5M DynamoDB Streams read request units.
	// https://aws.amazon.com/dynamodb/pricing/on-demand/
	t := time.NewTimer(0)
	defer t.Stop()

	currentShardIter := si.ShardIterator

loop:
	for {
		select {
		case <-ctx.Done():
			return nil

		case <-t.C:
			r, err := a.dyndbStrClient.GetRecordsWithContext(ctx, &dynamodbstreams.GetRecordsInput{
				ShardIterator: currentShardIter,
			})
			if err != nil {
				return fmt.Errorf("getting records from shard ID %s: %w", *shardID, err)
			}

			nextRequestDelay := getRecordsPeriod
			if len(r.Records) > 0 {
				// keep iterating immediately if any record was
				// returned, so that bursts of new records are
				// processed quickly
				nextRequestDelay = 0
			}

			for _, r := range r.Records {
				a.logger.Debug("Processing record ID: " + *r.EventID)

				if err := a.sendDynamoDBEvent(ctx, r); err != nil {
					return fmt.Errorf("sending CloudEvent: %w", err)
				}
			}

			currentShardIter = r.NextShardIterator

			// ShardIterator only becomes nil when the shard is
			// sealed (marked as READ_ONLY), which happens on
			// average every 4 hours.
			if currentShardIter == nil {
				a.logger.Debug("Shard ID ", *shardID, " got sealed")
				break loop
			}

			t.Reset(nextRequestDelay)
		}
	}

	return nil
}

// sendDynamoDBEvent sends the given Record as a CloudEvent.
func (a *adapter) sendDynamoDBEvent(ctx context.Context, r *dynamodbstreams.Record) error {
	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(v1alpha1.AWSEventType(a.arn.Service, v1alpha1.AWSDynamoDBGenericEventType))
	event.SetSubject(asEventSubject(r))
	event.SetSource(a.arn.String())
	event.SetID(*r.EventID)
	event.SetExtension(ceExtDynamoDBOperation, *r.EventName)
	if err := event.SetData(cloudevents.ApplicationJSON, r); err != nil {
		return fmt.Errorf("failed to set event data: %w", err)
	}

	if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		return result
	}
	return nil
}

// asEventSubject returns an event subject corresponding to the given record.
func asEventSubject(r *dynamodbstreams.Record) string {
	if r == nil || r.Dynamodb == nil || r.Dynamodb.Keys == nil {
		return ""
	}

	subject := strBuilderPool.Get().(*strings.Builder)
	defer strBuilderPool.Put(subject)
	defer subject.Reset()

	i := 0
	for k := range r.Dynamodb.Keys {
		subject.WriteString(k)
		i++
		if i < len(r.Dynamodb.Keys) {
			subject.WriteByte(',')
		}
	}

	return subject.String()
}

var strBuilderPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}
