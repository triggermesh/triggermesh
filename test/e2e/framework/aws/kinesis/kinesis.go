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

// Package kinesis contains helpers for AWS Kinesis.
package kinesis

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateDatastream creates a stream named after the given framework.Framework.
func CreateDatastream(kc kinesisiface.KinesisAPI, f *framework.Framework) string /*arn*/ {
	stream := &kinesis.CreateStreamInput{
		StreamName: &f.UniqueName,
		ShardCount: aws.Int64(1),
	}

	_, err := kc.CreateStream(stream)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create stream %q: %s", *stream.StreamName, err)
	}

	if err := kc.WaitUntilStreamExists(&kinesis.DescribeStreamInput{StreamName: stream.StreamName}); err != nil {
		framework.FailfWithOffset(2, "Failed while waiting for stream to exist: %s", err)
	}

	output, err := kc.DescribeStream(&kinesis.DescribeStreamInput{StreamName: stream.StreamName})
	if err != nil {
		framework.FailfWithOffset(2, "Failed to describe stream: %s", err)
	}

	return *output.StreamDescription.StreamARN
}

// PutRecord sends a message to a stream by name.
func PutRecord(kc kinesisiface.KinesisAPI, name string) string /*seqNumber*/ {
	params := &kinesis.PutRecordInput{
		Data:         []byte("hello, world!"),
		StreamName:   aws.String(name),
		PartitionKey: aws.String("key1"),
	}

	putOutput, err := kc.PutRecord(params)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to send message to stream: %s", err)
	}

	return *putOutput.SequenceNumber
}

// GetRecords get records from a kinesis data stream.
func GetRecords(kc kinesisiface.KinesisAPI, name string) []*kinesis.Record {
	var recordList []*kinesis.Record

	shards, err := kc.ListShards(&kinesis.ListShardsInput{
		StreamName: &name,
	})
	if err != nil {
		framework.FailfWithOffset(2, "Failed to get shards from stream: %s", err)
	}

	for _, s := range shards.Shards {
		shardIterator, err := kc.GetShardIterator(&kinesis.GetShardIteratorInput{
			ShardId:           s.ShardId,
			ShardIteratorType: aws.String("TRIM_HORIZON"),
			StreamName:        &name,
		})
		if err != nil {
			framework.FailfWithOffset(2, "Failed to get shard iterator from stream: %s", err)
		}

		records, err := kc.GetRecords(&kinesis.GetRecordsInput{
			ShardIterator: shardIterator.ShardIterator,
		})
		if err != nil {
			framework.FailfWithOffset(2, "Failed to get records from stream: %s", err)
		}
		recordList = append(recordList, records.Records...)
	}

	return recordList
}

// DeleteStream deletes a Kinesis stream by name.
func DeleteStream(kc kinesisiface.KinesisAPI, name string) {
	stream := &kinesis.DeleteStreamInput{
		StreamName: aws.String(name),
	}

	if _, err := kc.DeleteStream(stream); err != nil {
		framework.FailfWithOffset(2, "Failed to delete stream %q: %s", *stream.StreamName, err)
	}
}
