/*
Copyright 2020-2021 TriggerMesh Inc.

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
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams/dynamodbstreamsiface"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"
)

const (
	tTableArnResource        = "table/MyTable"
	tLatestStreamArnResource = "stream/1970-01-01T00:00:00.000"
	tShardIDPrefix           = "shardId-00000000000000000000-000000" // + 3 digits appended for each shard
)

func TestAdapter(t *testing.T) {
	// The test's data is pre-populated so the flow of records is
	// uninterrupted until every record has been retrieved. We can
	// therefore affirm something went wrong if the getRecords timer
	// happens during the test.
	const testTimeout = getRecordsPeriod

	const numShards = 3
	const itersPerShard = 3
	const expectEvents = numShards * itersPerShard * 3 // 3 hardcoded records per iterator

	ceClient := adaptertest.NewTestClient()

	strClient := &standardMockDynamoDBStreamsClient{
		shards: makeMockShards(numShards, itersPerShard),
	}

	a := adapter{
		logger:         loggingtesting.TestLogger(t),
		dyndbClient:    &standardMockDynamoDBClient{},
		dyndbStrClient: strClient,
		arn:            makeARN(tTableArnResource),
		ceClient:       ceClient,
	}

	testCtx, testCancel := context.WithTimeout(context.Background(), testTimeout)
	defer testCancel()

	startCtx, startCancel := context.WithCancel(testCtx)
	defer startCancel()

	errCh := make(chan error)
	defer close(errCh)

	go func() {
		errCh <- a.Start(startCtx)
	}()

	timer := time.NewTimer(0)
	defer timer.Stop()

poll:
	for {
		select {
		case <-testCtx.Done():
			t.Fatal("Timeout waiting for events")

		case <-timer.C:
			if len(ceClient.Sent()) >= expectEvents {
				startCancel()
				break poll
			}
			timer.Reset(5 * time.Millisecond)
		}
	}

	// no matter what, Start() should always return after its context has
	// been cancelled
	select {
	case <-testCtx.Done():
		t.Fatal("Timeout waiting for Start to return")

	case err := <-errCh:
		assert.NoError(t, err)
	}

	// final assertion to ensure we didn't receive more events than expected
	assert.Len(t, ceClient.Sent(), expectEvents)

	// asserting a single event suffices since the entire data set is mocked
	ev := ceClient.Sent()[0]
	assert.Equal(t, ev.Type(), "com.amazon.dynamodb.stream_record")
	assert.Equal(t, "arn:aws:dynamodb:us-fake-0:123456789012:table/MyTable", ev.Source())
	assert.Contains(t, []string{"id,name", "name,id"}, ev.Subject())
	assert.Contains(t, validDynamoDBOperations, ev.Extensions()[ceExtDynamoDBOperation])
}

// Enumerates valid DynamoDB operations / event names.
var validDynamoDBOperations = []string{
	dynamodbstreams.OperationTypeInsert,
	dynamodbstreams.OperationTypeModify,
	dynamodbstreams.OperationTypeRemove,
}

// makeARN returns a fake DynamoDB ARN for the given resource.
func makeARN(resource string) arn.ARN {
	return arn.ARN{
		Partition: "aws",
		Service:   "dynamodb",
		Region:    "us-fake-0",
		AccountID: "123456789012",
		Resource:  resource,
	}
}

// standardMockDynamoDBClient is a mocked DynamoDB client which returns a
// standard set of responses and never errors.
type standardMockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI
}

func (c *standardMockDynamoDBClient) DescribeTableWithContext(context.Context,
	*dynamodb.DescribeTableInput, ...request.Option) (*dynamodb.DescribeTableOutput, error) {

	latestStreamARN := makeARN(tLatestStreamArnResource).String()

	return &dynamodb.DescribeTableOutput{
		Table: &dynamodb.TableDescription{
			LatestStreamArn: aws.String(latestStreamARN),
		},
	}, nil
}

// standardMockDynamoDBStreamsClient is a mocked DynamoDBStreams client which
// returns a standard set of responses and never errors.
type standardMockDynamoDBStreamsClient struct {
	dynamodbstreamsiface.DynamoDBStreamsAPI

	shards mockShards
}

func (c *standardMockDynamoDBStreamsClient) DescribeStreamWithContext(context.Context,
	*dynamodbstreams.DescribeStreamInput, ...request.Option) (*dynamodbstreams.DescribeStreamOutput, error) {

	shards := make([]*dynamodbstreams.Shard, 0, len(c.shards))

	for id := range c.shards {
		shards = append(shards, &dynamodbstreams.Shard{
			ShardId: id,
		})
	}

	sort.Slice(shards, func(i, j int) bool {
		return *shards[i].ShardId < *shards[j].ShardId
	})

	return &dynamodbstreams.DescribeStreamOutput{
		StreamDescription: &dynamodbstreams.StreamDescription{
			StreamStatus: aws.String(dynamodbstreams.StreamStatusEnabled),
			Shards:       shards,
		},
	}, nil
}

func (c *standardMockDynamoDBStreamsClient) GetShardIteratorWithContext(_ context.Context,
	in *dynamodbstreams.GetShardIteratorInput, _ ...request.Option) (*dynamodbstreams.GetShardIteratorOutput, error) {

	return &dynamodbstreams.GetShardIteratorOutput{
		ShardIterator: c.shards[in.ShardId][0].name,
	}, nil
}

func (c *standardMockDynamoDBStreamsClient) GetRecordsWithContext(_ context.Context,
	in *dynamodbstreams.GetRecordsInput, _ ...request.Option) (*dynamodbstreams.GetRecordsOutput, error) {

	var recs []*dynamodbstreams.Record
	var nextIter *string

	for _, iters := range c.shards {
		for i, iter := range iters {
			if *iter.name == *in.ShardIterator {
				recs = iter.records

				// erase records to simulate "LATEST" shard
				// iterator type and prevent processing the
				// same record more than once
				iter.records = nil

				// loop over shard iterators infinitely to
				// simulate DynamoDB Streams API
				nextIter = iters[(i+1)%len(iters)].name

				break
			}
		}
	}

	return &dynamodbstreams.GetRecordsOutput{
		Records:           recs,
		NextShardIterator: nextIter,
	}, nil
}

// mockShards mocks the data contained in some DynamoDB Streams Shards. It
// represents the following structure:
//
// []shard id
//    \_ [] shard iterator
//           \_ [] record
//
type mockShards map[ /*shard id*/ *string][]*mockShardIterator
type mockShardIterator struct {
	name    *string
	records []*dynamodbstreams.Record
}

// makeMockShards returns a set of mocked Shards.
func makeMockShards(n, itersPerShard int) mockShards {
	shards := make(mockShards, n)

	for i := 0; i < n; i++ {
		id := aws.String(fmt.Sprintf(tShardIDPrefix+"%03d", i+1))
		shards[id] = makeMockIterators(itersPerShard, i+1)
	}

	return shards
}

// makeMockIterators returns a set of mocked ShardIterators for the given shard index.
func makeMockIterators(n, shardIdx int) []*mockShardIterator {
	// NOTE: shard iterators names are completely random in the "real" implementation

	latestStreamARN := makeARN(tLatestStreamArnResource).String()

	iters := make([]*mockShardIterator, n)

	for i := 0; i < n; i++ {
		iters[i] = &mockShardIterator{
			name:    aws.String(fmt.Sprintf(latestStreamARN+"|1|AAA...shard%03d...%03d", shardIdx, i+1)),
			records: makeMockRecords(shardIdx, i+1),
		}
	}

	return iters
}

// makeMockRecords returns a set of mocked StreamRecords for the given shard
// and iterator indexes (exactly 3, to keep the data set simple and predictable).
func makeMockRecords(shardIdx, iteratorIdx int) []*dynamodbstreams.Record {
	// NOTE: event IDs are completely random in the "real" implementation

	return []*dynamodbstreams.Record{{
		EventID:   aws.String(fmt.Sprintf("shard%03d-iterator%03d-001", shardIdx, iteratorIdx)),
		EventName: aws.String(dynamodbstreams.OperationTypeInsert),
		Dynamodb: &dynamodbstreams.StreamRecord{
			Keys: map[string]*dynamodb.AttributeValue{"id": nil, "name": nil},
		},
	}, {
		EventID:   aws.String(fmt.Sprintf("shard%03d-iterator%03d-002", shardIdx, iteratorIdx)),
		EventName: aws.String(dynamodbstreams.OperationTypeModify),
		Dynamodb: &dynamodbstreams.StreamRecord{
			Keys: map[string]*dynamodb.AttributeValue{"id": nil, "name": nil},
		},
	}, {
		EventID:   aws.String(fmt.Sprintf("shard%03d-iterator%03d-003", shardIdx, iteratorIdx)),
		EventName: aws.String(dynamodbstreams.OperationTypeRemove),
		Dynamodb: &dynamodbstreams.StreamRecord{
			Keys: map[string]*dynamodb.AttributeValue{"id": nil, "name": nil},
		},
	}}
}
