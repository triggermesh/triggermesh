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

// Package sns contains helpers for AWS SNS.
package sns

import (
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateTopic creates a topic named after the given framework.Framework.
func CreateTopic(snsClient snsiface.SNSAPI, f *framework.Framework) string /*arn*/ {
	topic := &sns.CreateTopicInput{
		Name: &f.UniqueName,
	}

	resp, err := snsClient.CreateTopic(topic)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create topic %q: %s", *topic.Name, err)
	}

	return *resp.TopicArn
}

// DeleteTopic deletes the topic with the given ARN.
func DeleteTopic(snsClient snsiface.SNSAPI, arn string) {
	topic := &sns.DeleteTopicInput{
		TopicArn: &arn,
	}

	if _, err := snsClient.DeleteTopic(topic); err != nil {
		framework.FailfWithOffset(2, "Failed to delete topic %q: %s", *topic.TopicArn, err)
	}
}

// SendMessage sends a message to the topic with the given ARN.
func SendMessage(snsClient snsiface.SNSAPI, arn string) string /*msgId*/ {
	msg := "hello, world!"

	params := &sns.PublishInput{
		TargetArn: &arn,
		Message:   &msg,
	}

	msgOutput, err := snsClient.Publish(params)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to publish message to topic %q: %s", *params.TargetArn, err)
	}
	return *msgOutput.MessageId
}
