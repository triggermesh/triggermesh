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

package awskinesissource

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"

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

	ARN string `envconfig:"ARN" required:"true"`

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

	knsClient kinesisiface.KinesisAPI
	ceClient  cloudevents.Client

	arn    arn.ARN
	stream string
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.AWSKinesisSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	arn := common.MustParseARN(env.ARN)

	cfg := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(arn.Region).
		WithMaxRetries(5),
	))

	return &adapter{
		logger: logger,
		mt:     mt,

		knsClient: kinesis.New(cfg),
		ceClient:  ceClient,

		arn:    arn,
		stream: common.MustParseKinesisResource(arn.Resource),
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	myStream, err := a.knsClient.DescribeStream(&kinesis.DescribeStreamInput{
		StreamName: &a.stream,
	})
	if err != nil {
		return fmt.Errorf("describing stream %q: %w", a.arn, err)
	}

	health.MarkReady()

	streamARN := myStream.StreamDescription.StreamARN

	a.logger.Infof("Connected to Kinesis stream: %s", *streamARN)

	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	// Obtain records inputs for different shards
	inputs := a.getRecordsInputs(myStream.StreamDescription.Shards)

	backoff := common.NewBackoff()

	err = backoff.Run(ctx.Done(), func(ctx context.Context) (bool, error) {
		resetBackoff := false
		records, err := a.processInputs(inputs)
		if err != nil {
			a.logger.Errorw("There were errors during inputs processing", zap.Error(err))
		}

		for _, record := range records {
			resetBackoff = true
			err = a.sendKinesisRecord(ctx, record)
			if err != nil {
				a.logger.Errorw("Failed to send cloudevent", zap.Error(err))
			}
		}
		return resetBackoff, nil
	})

	return err
}

func (a *adapter) getRecordsInputs(shards []*kinesis.Shard) []kinesis.GetRecordsInput {
	inputs := []kinesis.GetRecordsInput{}

	// Kinesis stream might have several shards and each of them had "LATEST" Iterator.
	for _, shard := range shards {
		// Obtain starting Shard Iterator. This is needed to not process already processed records
		myShardIterator, err := a.knsClient.GetShardIterator(&kinesis.GetShardIteratorInput{
			ShardId:           shard.ShardId,
			ShardIteratorType: aws.String("LATEST"),
			StreamName:        &a.stream,
		})

		if err != nil {
			a.logger.Errorw("Failed to get shard iterator", zap.Error(err))
			continue
		}

		// set records output limit. Should not be more than 10000, othervise panics
		input := kinesis.GetRecordsInput{
			ShardIterator: myShardIterator.ShardIterator,
		}

		inputs = append(inputs, input)
	}

	return inputs
}

func (a *adapter) processInputs(inputs []kinesis.GetRecordsInput) ([]*kinesis.Record, error) {
	var errs []error
	records := []*kinesis.Record{}

	for i, input := range inputs {
		input := input

		recordsOutput, err := a.knsClient.GetRecords(&input)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		records = append(records, recordsOutput.Records...)

		// remove old input
		inputs = append(inputs[:i], inputs[i+1:]...)

		// generate new input
		input = kinesis.GetRecordsInput{
			ShardIterator: recordsOutput.NextShardIterator,
		}

		// add newly generated input to the slice
		// so that new iteration would begin with new sharIterator
		inputs = append(inputs, input)
	}

	return records, utilerrors.NewAggregate(errs)
}

func (a *adapter) sendKinesisRecord(ctx context.Context, record *kinesis.Record) error {
	a.logger.Infof("Processing record ID: %s", *record.SequenceNumber)

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(v1alpha1.AWSEventType(a.arn.Service, v1alpha1.AWSKinesisGenericEventType))
	event.SetSubject(*record.PartitionKey)
	event.SetSource(a.arn.String())
	event.SetID(*record.SequenceNumber)
	if err := event.SetData(cloudevents.ApplicationJSON, toCloudEventData(record)); err != nil {
		return fmt.Errorf("failed to set event data: %w", err)
	}

	if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		return result
	}
	return nil
}

// toCloudEventData returns a Kinesis record in a shape that is suitable for
// JSON serialization inside some CloudEvent data.
func toCloudEventData(record *kinesis.Record) interface{} {
	var data interface{}
	data = record

	// if record.Data contains raw JSON data, type it as json.RawMessage so
	// it doesn't get encoded to base64 during the serialization of the
	// CloudEvent data.
	if json.Valid(record.Data) {
		data = &RecordWithRawJSONData{
			Data:   json.RawMessage(record.Data),
			Record: record,
		}
	}

	return data
}

// RecordWithRawJSONData is an Message with RawMessage-typed JSON data.
type RecordWithRawJSONData struct {
	Data json.RawMessage
	*kinesis.Record
}
