/*
Copyright 2021 TriggerMesh Inc.

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

// Package sqs contains helpers for AWS SQS.
package sqs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	e2eaws "github.com/triggermesh/triggermesh/test/e2e/framework/aws"
)

// CreateQueue creates a queue named after the given framework.Framework.
func CreateQueue(sqsClient sqsiface.SQSAPI, f *framework.Framework) string /*url*/ {
	queue := &sqs.CreateQueueInput{
		QueueName: &f.UniqueName,
		Tags:      e2eaws.TagsFor(f),
	}

	resp, err := sqsClient.CreateQueue(queue)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create queue %q: %s", *queue.QueueName, err)
	}

	return *resp.QueueUrl
}

// DeleteQueue deletes the queue with the given URL.
func DeleteQueue(sqsClient sqsiface.SQSAPI, url string) {
	queue := &sqs.DeleteQueueInput{
		QueueUrl: &url,
	}

	if _, err := sqsClient.DeleteQueue(queue); err != nil {
		framework.FailfWithOffset(2, "Failed to delete queue %q: %s", *queue.QueueUrl, err)
	}
}

// QueueARN returns the ARN of the queue with the given URL.
func QueueARN(sqsClient sqsiface.SQSAPI, url string) string /*arn*/ {
	attribs := &sqs.GetQueueAttributesInput{
		QueueUrl: &url,
		AttributeNames: aws.StringSlice([]string{
			sqs.QueueAttributeNameQueueArn,
		}),
	}

	resp, err := sqsClient.GetQueueAttributes(attribs)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to get attributes of queue %q: %s", *attribs.QueueUrl, err)
	}

	return *resp.Attributes[sqs.QueueAttributeNameQueueArn]
}

// SendMessage sends a message to the queue with the given URL.
func SendMessage(sqsClient sqsiface.SQSAPI, url string) string /*msgId*/ {
	msg := "hello, world!"

	params := &sqs.SendMessageInput{
		QueueUrl:    &url,
		MessageBody: &msg,
	}

	msgOutput, err := sqsClient.SendMessage(params)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to send message to queue %q: %s", *params.QueueUrl, err)
	}
	return *msgOutput.MessageId
}
