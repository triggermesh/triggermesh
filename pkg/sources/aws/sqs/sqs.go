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

// Package sqs contains helpers for AWS SQS.
package sqs

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	"github.com/triggermesh/triggermesh/pkg/sources/aws/iam"
)

// CreateQueue creates a queue with the given name and optional tags.
//
// Naming restrictions are described at https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference/API_CreateQueue.html
func CreateQueue(cli sqsiface.SQSAPI, name string, tags map[string]string) (string /*url*/, error) {
	queue := &sqs.CreateQueueInput{
		QueueName: &name,
		Tags:      aws.StringMap(tags),
	}

	resp, err := cli.CreateQueue(queue)
	if err != nil {
		return "", fmt.Errorf("creating queue %q: %w", *queue.QueueName, err)
	}

	return *resp.QueueUrl, nil
}

// SetQueuePolicy sets the Policy attribute of the queue with the given URL.
//
// See also https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-authentication-and-access-control.html
func SetQueuePolicy(cli sqsiface.SQSAPI, url string, pol iam.Policy) error {
	polJSON, err := json.Marshal(pol)
	if err != nil {
		return fmt.Errorf("serializing queue policy to JSON: %w", err)
	}

	attrs := &sqs.SetQueueAttributesInput{
		QueueUrl: &url,
		Attributes: aws.StringMap(map[string]string{
			sqs.QueueAttributeNamePolicy: string(polJSON),
		}),
	}

	if _, err := cli.SetQueueAttributes(attrs); err != nil {
		return fmt.Errorf("setting attributes of queue %q: %w", *attrs.QueueUrl, err)
	}

	return nil
}

// DeleteQueue deletes the queue with the given URL.
func DeleteQueue(cli sqsiface.SQSAPI, url string) error {
	queue := &sqs.DeleteQueueInput{
		QueueUrl: &url,
	}

	if _, err := cli.DeleteQueue(queue); err != nil {
		return fmt.Errorf("deleting queue %q: %w", *queue.QueueUrl, err)
	}

	return nil
}

// QueuePolicy returns the policy of the queue with the given URL.
func QueuePolicy(cli sqsiface.SQSAPI, url string) (string /*policy*/, error) {
	attribs := &sqs.GetQueueAttributesInput{
		QueueUrl: &url,
		AttributeNames: aws.StringSlice([]string{
			sqs.QueueAttributeNamePolicy,
		}),
	}

	resp, err := cli.GetQueueAttributes(attribs)
	if err != nil {
		return "", fmt.Errorf("getting attributes of queue %q: %w", *attribs.QueueUrl, err)
	}

	return *resp.Attributes[sqs.QueueAttributeNamePolicy], nil
}

// QueueARN returns the ARN of the queue with the given URL.
func QueueARN(cli sqsiface.SQSAPI, url string) (string /*arn*/, error) {
	attrs, err := QueueAttributes(cli, url, []string{sqs.QueueAttributeNameQueueArn})
	if err != nil {
		return "", fmt.Errorf("getting ARN attribute: %w", err)
	}

	return attrs[sqs.QueueAttributeNameQueueArn], nil
}

// QueueAttributes returns selected attributes of the queue with the given URL.
func QueueAttributes(cli sqsiface.SQSAPI, url string, attrs []string) (map[string]string, error) {
	attribs := &sqs.GetQueueAttributesInput{
		QueueUrl:       &url,
		AttributeNames: aws.StringSlice(attrs),
	}

	resp, err := cli.GetQueueAttributes(attribs)
	if err != nil {
		return nil, fmt.Errorf("getting attributes of queue %q: %w", *attribs.QueueUrl, err)
	}

	return aws.StringValueMap(resp.Attributes), nil
}

// QueueURL returns the URL of the queue identified by name.
func QueueURL(cli sqsiface.SQSAPI, name string) (string /*url*/, error) {
	queue := &sqs.GetQueueUrlInput{
		QueueName: &name,
	}

	resp, err := cli.GetQueueUrl(queue)
	if err != nil {
		return "", fmt.Errorf("getting URL of queue %q: %w", *queue.QueueName, err)
	}

	return *resp.QueueUrl, nil
}

// QueueTags returns the tags of the queue with the given URL.
func QueueTags(cli sqsiface.SQSAPI, url string) (map[string]string, error) {
	queue := &sqs.ListQueueTagsInput{
		QueueUrl: &url,
	}

	resp, err := cli.ListQueueTags(queue)
	if err != nil {
		return nil, fmt.Errorf("listing tags of queue %q: %w", *queue.QueueUrl, err)
	}

	return aws.StringValueMap(resp.Tags), nil
}
