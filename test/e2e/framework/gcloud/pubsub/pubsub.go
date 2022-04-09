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

// Package pubsub contains helpers for Google Cloud Pub/Sub.
package pubsub

import (
	"context"

	"cloud.google.com/go/pubsub"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/gcloud"
)

// CreateTopic creates a Pub/Sub topic named after the given framework.Framework.
func CreateTopic(pubsubCli *pubsub.Client, f *framework.Framework) *pubsub.Topic {
	topicID := f.UniqueName

	cfg := &pubsub.TopicConfig{
		Labels: gcloud.TagsFor(f),
	}

	topic, err := pubsubCli.CreateTopicWithConfig(context.Background(), topicID, cfg)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create topic %q: %s", topicID, err)
	}

	return topic
}

// CreateSubscription creates a subscription for the given Pub/Sub topic.
func CreateSubscription(pubsubCli *pubsub.Client, topic *pubsub.Topic, f *framework.Framework) *pubsub.Subscription {
	cfg := pubsub.SubscriptionConfig{
		Topic:  topic,
		Labels: gcloud.TagsFor(f),
	}

	subscriptionID := topic.ID()

	subscription, err := pubsubCli.CreateSubscription(context.Background(), subscriptionID, cfg)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create subscription for topic %q: %s", topic.ID(), err)
	}

	return subscription
}

// SendMessage sends a message to the given Pub/Sub topic and returns the
// corresponding message ID.
func SendMessage(pubsubCli *pubsub.Client, topicID string) string /*msg id*/ {
	msg := &pubsub.Message{
		Data: []byte("Hello, World!"),
	}

	ctx := context.Background()

	publishResult := pubsubCli.Topic(topicID).Publish(ctx, msg)

	msgID, err := publishResult.Get(ctx)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to send message to topic %q: %s", topicID, err)
	}

	return msgID
}

// DeleteTopic deletes the Pub/Sub topic with the given name.
func DeleteTopic(pubsubCli *pubsub.Client, topicID string) {
	if err := pubsubCli.Topic(topicID).Delete(context.Background()); err != nil {
		framework.FailfWithOffset(2, "Failed to delete topic %q: %s", topicID, err)
	}
}

// DeleteSubscription deletes the Pub/Sub subscription with the given name.
func DeleteSubscription(pubsubCli *pubsub.Client, subsID string) {
	if err := pubsubCli.Subscription(subsID).Delete(context.Background()); err != nil {
		framework.FailfWithOffset(2, "Failed to delete subscription %q: %s", subsID, err)
	}
}
