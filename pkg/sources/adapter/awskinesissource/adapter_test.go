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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	testCases := []struct {
		name            string
		eventData       []byte
		expectEventData interface{}
	}{
		{
			name:            "Data is raw bytes",
			eventData:       []byte{'t', 'e', 's', 't'},
			expectEventData: `"dGVzdA=="`, // base64-encoded "test"
		},
		{
			name:            "Data is a JSON object",
			eventData:       []byte(`{"test": null}`),
			expectEventData: `{"test":null}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ceClient := adaptertest.NewTestClient()

			a := &adapter{
				logger:   loggingtesting.TestLogger(t),
				stream:   "fooStream",
				ceClient: ceClient,
			}

			record := kinesis.Record{
				Data:           tc.eventData,
				SequenceNumber: aws.String("1"),
				PartitionKey:   aws.String("key"),
			}

			ctx := context.Background()

			err := a.sendKinesisRecord(ctx, &record)
			assert.NoError(t, err)

			gotEvents := ceClient.Sent()
			require.Len(t, gotEvents, 1, "Expected 1 event")

			eventData := make(map[string]interface{})
			err = json.Unmarshal(gotEvents[0].Data(), &eventData)
			require.NoError(t, err)

			assert.Equal(t, "1", eventData["SequenceNumber"])
			assert.Equal(t, "key", eventData["PartitionKey"])

			// ensure the sent event has the expected encoding (base64 / raw JSON)
			eventDataStr := stringifyEventData(t, eventData["Data"])
			assert.Equal(t, tc.expectEventData, eventDataStr)
		})
	}
}

// stringifyEventData returns the given data as a JSON-encoded string.
// This helps asserting the value of a Kinesis record's Data contained in a
// CloudEvent, which can be either a JSON object encoded as a
// map[string]interface{} (if the original record contained JSON data) or a
// base64-encoded string (for any other type of data).
func stringifyEventData(t *testing.T, data interface{}) string {
	dataBytes, err := json.Marshal(data)
	require.NoError(t, err)

	var jsonData json.RawMessage
	err = json.Unmarshal(dataBytes, &jsonData)
	require.NoError(t, err)

	return string(jsonData)
}
