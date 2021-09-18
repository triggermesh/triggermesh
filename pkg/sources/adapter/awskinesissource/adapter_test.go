/*
Copyright 2019-2020 TriggerMesh Inc.

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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"
)

type mockedGetRecords struct {
	kinesisiface.KinesisAPI
	Resp kinesis.GetRecordsOutput
	err  error
}

type mockedGetShardIterator struct {
	kinesisiface.KinesisAPI
	Resp kinesis.GetShardIteratorOutput
	err  error
}

func (m mockedGetRecords) GetRecords(in *kinesis.GetRecordsInput) (*kinesis.GetRecordsOutput, error) {
	return &m.Resp, m.err
}

func (m mockedGetShardIterator) GetShardIterator(in *kinesis.GetShardIteratorInput) (*kinesis.GetShardIteratorOutput, error) {
	return &m.Resp, m.err
}

func TestProcessInputs(t *testing.T) {
	now := time.Now()
	records := []*kinesis.Record{
		{
			SequenceNumber:              aws.String("1"),
			PartitionKey:                aws.String("key"),
			ApproximateArrivalTimestamp: &now,
			Data:                        []byte("foo"),
		},
	}

	a := &adapter{
		logger:   loggingtesting.TestLogger(t),
		ceClient: adaptertest.NewTestClient(),
		stream:   "arn:aws:kinesis:us-east-1:123456789012:stream/foo",
	}

	a.knsClient = mockedGetRecords{
		Resp: kinesis.GetRecordsOutput{
			NextShardIterator: aws.String("nextIterator"),
			Records:           records,
		},
		err: nil,
	}

	inputs := []kinesis.GetRecordsInput{
		{},
	}

	_, err := a.processInputs(inputs)
	assert.NoError(t, err)

	const errMsg = "fake error"

	a.knsClient = mockedGetRecords{
		Resp: kinesis.GetRecordsOutput{},
		err:  errors.New(errMsg),
	}

	_, err = a.processInputs(inputs)
	assert.EqualError(t, err, errMsg)
}

func TestGetRecordsInputs(t *testing.T) {
	a := &adapter{
		logger: loggingtesting.TestLogger(t),
	}

	a.knsClient = mockedGetShardIterator{
		Resp: kinesis.GetShardIteratorOutput{ShardIterator: aws.String("shardIterator")},
		err:  nil,
	}

	shards := []*kinesis.Shard{
		{ShardId: aws.String("1")},
	}

	inputs := a.getRecordsInputs(shards)
	assert.Equal(t, 1, len(inputs))

	a.knsClient = mockedGetShardIterator{
		Resp: kinesis.GetShardIteratorOutput{},
		err:  errors.New("fake error"),
	}

	inputs = a.getRecordsInputs(shards)
	assert.Equal(t, 0, len(inputs))
}

func TestSendCloudevent(t *testing.T) {
	ceClient := adaptertest.NewTestClient()

	a := &adapter{
		logger:   loggingtesting.TestLogger(t),
		stream:   "fooStream",
		ceClient: ceClient,
	}

	record := kinesis.Record{
		Data:           []byte("foo"),
		SequenceNumber: aws.String("1"),
		PartitionKey:   aws.String("key"),
	}

	err := a.sendKinesisRecord(&record)
	assert.NoError(t, err)

	gotEvents := ceClient.Sent()
	assert.Len(t, gotEvents, 1, "Expected 1 event, got %d", len(gotEvents))

	var gotData kinesis.Record
	err = gotEvents[0].DataAs(&gotData)
	assert.NoError(t, err)
	assert.EqualValues(t, record, gotData, "Expected event %q, got %q", record, gotData)
}
