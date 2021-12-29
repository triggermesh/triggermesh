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

// Package pubsub contains helpers for Google Cloud PubSub.
package pubsub

import (
	"context"

	"cloud.google.com/go/pubsub"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/gcloud"
)

// CreateTopic creates a topic named after the given framework.Framework.
func CreateTopic(pubsubCli *pubsub.Client, f *framework.Framework) *pubsub.Topic {
	cfg := &pubsub.TopicConfig{
		Labels: gcloud.TagsFor(f),
	}

	topicID := gcloud.PubSubResourceID(f)

	resp, err := pubsubCli.CreateTopicWithConfig(context.Background(), topicID, cfg)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create topic %q: %s", topicID, err)
	}

	return resp
}

// CreateSubscription creates a subscription for a topic.
func CreateSubscription(pubsubCli *pubsub.Client, topic *pubsub.Topic, f *framework.Framework) string {
	cfg := pubsub.SubscriptionConfig{
		Topic:  topic,
		Labels: gcloud.TagsFor(f),
	}

	subscriptionID := gcloud.PubSubResourceID(f)

	resp, err := pubsubCli.CreateSubscription(context.Background(), subscriptionID, cfg)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create subscription for topic %q: %s", topic.String(), err)
	}

	return resp.String()
}

func SendMessage(pubsubCli *pubsub.Client, topic *pubsub.Topic, f *framework.Framework) string {
	sendMessage := pubsubCli.Topic(topic.ID()).Publish(context.Background(), &pubsub.Message{
		Data: []byte("Hello world"),
	})

	id, err := sendMessage.Get(context.Background())
	if err != nil {
		framework.FailfWithOffset(2, "Failed to send message: %s", err)
	}

	return id
}

func DeleteTopic(pubsubCli *pubsub.Client, topic *pubsub.Topic) {
	err := pubsubCli.Topic(topic.ID()).Delete(context.Background())
	if err != nil {
		framework.FailfWithOffset(2, "Failed to delete topic %q: %s", topic, err)
	}
}
