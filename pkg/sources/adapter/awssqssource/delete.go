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

package awssqssource

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

const (
	// Highest possible value for the MaxNumberOfMessages request parameter.
	// Undocumented, but exceeding that number returns "Maximum number of
	// entries per request are 10. You have sent ...".
	maxDeleteMsgBatchSize = 10

	// Maximum time to wait between calls to DeleteMessageBatch, which
	// marks messages as processed.
	maxDeleteMsgPeriod = 3 * time.Second

	// Calls to DeleteMessage are cancelled when they exceed this duration.
	deleteRequestTimeout = 10 * time.Second
)

// A message deleter deletes messages from the SQS queue to mark them as
// processed. It does this by accumulating references of SQS messages into a
// deletion buffer until this buffer has reached its capacity or until a timer
// expires, whichever happens first.
func (a *adapter) runMessagesDeleter(ctx context.Context, queueURL string) {
	delMsgBuf := make(messageDeleteBuffer, maxDeleteMsgBatchSize)

	t := time.NewTimer(a.deletePeriod)

	// calling this function blocks the processing of received messages by
	// this deleter temporarily
	handleDeletion := func() {
		defer t.Reset(a.deletePeriod)

		if len(delMsgBuf) == 0 {
			return
		}

		a.logger.Debugw("Deleting messages", zap.Array(logfieldMsgIDs, delMsgBuf))

		if err := deleteMessages(ctx, a.sqsClient, queueURL, delMsgBuf); err != nil {
			// NOTE(antoineco): If the batch deletion fails, SQS
			// will re-add those messages to the queue after the
			// visibility timeout has expired, causing a
			// re-delivery (at-least-once delivery).
			a.logger.Errorw("Failed to delete messages from the SQS queue", zap.Error(err))
		}

		// reuse the same buffer to avoid new allocations
		for k := range delMsgBuf {
			delete(delMsgBuf, k)
		}
	}

	for {
		select {
		case <-ctx.Done():
			// always flush current message buffer upon termination
			ctx = context.Background()
			handleDeletion()

			return

		case <-t.C:
			handleDeletion()

		case msg := <-a.deleteQueue:
			a.sr.reportMessageDequeuedDeleteCount()

			delMsgBuf[*msg.MessageId] = *msg.ReceiptHandle

			if len(delMsgBuf) == maxDeleteMsgBatchSize {
				handleDeletion()
			}
		}
	}
}

// messageDeleteBuffer holds references to SQS messages that have already been
// processed and should be deleted from the SQS queue.
type messageDeleteBuffer map[ /*MessageId*/ string] /*ReceiptHandle*/ string

var _ zapcore.ArrayMarshaler = (messageDeleteBuffer)(nil)

// MarshalLogArray implements zapcore.ArrayMarshaler.
func (mdb messageDeleteBuffer) MarshalLogArray(arr zapcore.ArrayEncoder) error {
	for id := range mdb {
		arr.AppendString(id)
	}
	return nil
}

// deleteMessages deletes messages from the SQS queue.
func deleteMessages(ctx context.Context, cli sqsiface.SQSAPI, queueURL string, msgs messageDeleteBuffer) error {
	deleteEntries := make([]*sqs.DeleteMessageBatchRequestEntry, 0, len(msgs))
	for id, handle := range msgs {
		deleteEntries = append(deleteEntries, &sqs.DeleteMessageBatchRequestEntry{
			Id:            aws.String(id),
			ReceiptHandle: aws.String(handle),
		})
	}

	ctx, cancel := context.WithTimeout(ctx, deleteRequestTimeout)
	defer cancel()

	in := &sqs.DeleteMessageBatchInput{
		QueueUrl: &queueURL,
		Entries:  deleteEntries,
	}

	out, err := cli.DeleteMessageBatchWithContext(ctx, in)
	if err != nil {
		return err
	}
	if len(out.Failed) > 0 {
		return errors.New(prettifyBatchResultErrors(out.Failed))
	}

	return nil
}
