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

package googlecloudstoragesource

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"cloud.google.com/go/pubsub"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
)

const (
	pubsubCollectionTopics = "topics"
	pubsubCollectionSubs   = "subscriptions"

	pubsubLabelOwnerResource  = "io-triggermesh_owner-resource"
	pubsubLabelOwnerNamespace = "io-triggermesh_owner-namespace"
	pubsubLabelOwnerName      = "io-triggermesh_owner-name"
)

// EnsurePubSub ensures the existence of a Pub/Sub topic and associated
// subscription for receiving change notifications from a Cloud Storage bucket.
func EnsurePubSub(ctx context.Context, cli *pubsub.Client) (*v1alpha1.GCloudResourceName /*topic*/, error) {
	if skip.Skip(ctx) {
		return nil, nil
	}

	topic, err := ensurePubSubTopic(ctx, cli)
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile Pub/Sub topic: %w", err)
	}

	if err := ensurePubSubSubscription(ctx, cli, topic); err != nil {
		return nil, fmt.Errorf("failed to reconcile Pub/Sub subscription: %w", err)
	}

	return topic, nil
}

// EnsureNoPubSub ensures the Pub/Sub topic and associated subscription used
// for receiving change notifications from a Cloud Storage bucket are deleted.
func EnsureNoPubSub(ctx context.Context, cli *pubsub.Client) error {
	if skip.Skip(ctx) {
		return nil
	}

	if err := ensureNoPubSubSubscription(ctx, cli); err != nil {
		return fmt.Errorf("failed to clean up Pub/Sub subscription: %w", err)
	}

	if err := ensureNoPubSubTopic(ctx, cli); err != nil {
		return fmt.Errorf("failed to clean up Pub/Sub topic: %w", err)
	}

	return nil
}

// ensurePubSubTopic ensures the existence of a Pub/Sub topic for receiving
// change notifications from the Cloud Storage bucket.
// Required permissions:
// - pubsub.topics.get
// - pubsub.topics.create
func ensurePubSubTopic(ctx context.Context, cli *pubsub.Client) (*v1alpha1.GCloudResourceName, error) {
	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudStorageSource)
	status := &src.Status

	if userProvided := src.Spec.PubSub.Topic; userProvided != nil {
		topicResName := userProvided

		// safeguard, in case the OpenAPI validation let a bad resource name through
		if userProvided.Collection != pubsubCollectionTopics {
			status.MarkNotSubscribed(v1alpha1.ReasonFailedSync, "Not a topic: "+topicResName.String())
			return nil, controller.NewPermanentError(
				reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
					"The provided resource name %q does not represent a Pub/Sub topic", topicResName),
			)
		}

		topic := cli.Topic(userProvided.Resource)

		exists, err := topic.Exists(ctx)
		switch {
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Access denied to Pub/Sub API: "+toErrMsg(err))
			return nil, controller.NewPermanentError(failGetTopicEvent(topicResName.String(), err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Cannot look up topic: "+toErrMsg(err))
			// wrap any other error to fail the reconciliation
			return nil, fmt.Errorf("%w", failGetTopicEvent(topicResName.String(), err))
		}

		if !exists {
			status.MarkNotSubscribed(v1alpha1.ReasonFailedSync,
				"Provided topic does not exist: "+topicResName.String())
			return nil, controller.NewPermanentError(
				reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
					"The provided topic %q does not exist", topicResName),
			)
		}

		status.Topic = topicResName

		return userProvided, nil
	}

	topicID := pubsubResourceID(src)

	topic := cli.Topic(topicID)

	exists, err := topic.Exists(ctx)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Pub/Sub API: "+toErrMsg(err))
		return nil, controller.NewPermanentError(failGetTopicEvent(topic.String(), err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Cannot look up topic: "+toErrMsg(err))
		// wrap any other error to fail the reconciliation
		return nil, fmt.Errorf("%w", failGetTopicEvent(topic.String(), err))
	}

	if !exists {
		cfg := &pubsub.TopicConfig{
			Labels: pubsubResourceLabels(src),
		}

		_, err = cli.CreateTopicWithConfig(ctx, topicID, cfg)
		switch {
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Access denied to Pub/Sub API: "+toErrMsg(err))
			return nil, controller.NewPermanentError(failCreateTopicEvent(topicID, err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Cannot create topic: "+toErrMsg(err))
			// wrap any other error to fail the reconciliation
			return nil, fmt.Errorf("%w", failCreateTopicEvent(topicID, err))
		}

		event.Normal(ctx, ReasonSubscribed, "Created topic %q", topicID)
	}

	topicResName := &v1alpha1.GCloudResourceName{}
	if err := json.Unmarshal([]byte(strconv.Quote(topic.String())), topicResName); err != nil {
		return nil, fmt.Errorf("failed to deserialize topic name: %w", err)
	}

	status.Topic = topicResName

	return topicResName, nil
}

// ensureNoPubSubTopic ensures that the Pub/Sub topic created for receiving
// messages from the Cloud Storage bucket is deleted.
// Required permissions:
// - pubsub.topics.get
// - pubsub.topics.delete
func ensureNoPubSubTopic(ctx context.Context, cli *pubsub.Client) error {
	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudStorageSource)
	status := src.Status

	if src.Spec.PubSub.Topic != nil {
		// do not delete topics managed by the user
		return nil
	}

	topicResName := status.Topic
	if topicResName == nil {
		// topic was possibly never created
		return nil
	}

	topic := cli.Topic(topicResName.Resource)

	cfg, err := topic.Config(ctx)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to Pub/Sub API. Ignoring: %s", toErrMsg(err))
		return nil
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed,
			fmt.Sprintf("Topic %q not found, skipping deletion", topic))
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Failed to get Pub/Sub topic %q: %s", topic, toErrMsg(err))
	}

	if !assertPubSubTopicOwnership(src, cfg) {
		event.Warn(ctx, ReasonUnsubscribed, "Topic %q is not owned by this source instance. "+
			"Skipping deletion", topic)
		return nil
	}

	err = topic.Delete(ctx)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to Pub/Sub API. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return failDeleteTopicEvent(topic.String(), err)
	}

	// do not return a reconciler.Event here, even of type "Normal",
	// otherwise the caller assumes an error occured
	event.Normal(ctx, ReasonUnsubscribed, "Deleted topic %q", topic)

	return nil
}

// ensurePubSubSubscription ensures the existence of a Pub/Sub subscription for
// receiving messages from the desired topic.
// Required permissions:
// - pubsub.subscriptions.get
// - pubsub.subscriptions.create
// - pubsub.topics.attachSubscription
func ensurePubSubSubscription(ctx context.Context, cli *pubsub.Client, topicResName *v1alpha1.GCloudResourceName) error {
	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudStorageSource)
	status := &src.Status

	// safeguard, in case the function is called with a bad resource name
	if topicResName.Collection != pubsubCollectionTopics {
		status.MarkNotSubscribed(v1alpha1.ReasonFailedSync, "Not a topic: "+topicResName.String())
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"The provided resource name %q does not represent a Pub/Sub topic", topicResName))
	}

	subsID := pubsubResourceID(src)

	subs := cli.Subscription(subsID)

	exists, err := subs.Exists(ctx)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Pub/Sub API: "+toErrMsg(err))
		return controller.NewPermanentError(failGetSubscriptionEvent(topicResName.String(), err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Cannot look up subscription: "+toErrMsg(err))
		return fmt.Errorf("%w", failGetSubscriptionEvent(topicResName.String(), err))
	}

	if !exists {
		cfg := pubsub.SubscriptionConfig{
			Topic:  cli.Topic(topicResName.Resource),
			Labels: pubsubResourceLabels(src),
		}

		_, err := cli.CreateSubscription(ctx, subsID, cfg)
		switch {
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Access denied to Pub/Sub API: "+toErrMsg(err))
			return controller.NewPermanentError(failCreateSubscriptionEvent(topicResName.String(), err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Cannot subscribe to topic: "+toErrMsg(err))
			return fmt.Errorf("%w", failCreateSubscriptionEvent(topicResName.String(), err))
		}
	}

	// it is essential that we propagate the subscription's name
	// here, otherwise BuildAdapter() won't be able to configure
	// the Pub/Sub adapter properly
	status.Subscription = makeSubscriptionResourceName(topicResName.Project, subsID)

	if !status.GetCondition(v1alpha1.GoogleCloudStorageConditionSubscribed).IsTrue() {
		event.Normal(ctx, ReasonSubscribed, "Subscribed to topic %q", topicResName)
	}

	return nil
}

// ensureNoPubSubSubscription ensures that the Pub/Sub subscription created for
// receiving messages from the desired topic is deleted.
// Required permissions:
// - pubsub.subscriptions.get
// - pubsub.subscriptions.delete
func ensureNoPubSubSubscription(ctx context.Context, cli *pubsub.Client) error {
	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudStorageSource)
	status := src.Status

	subsResName := status.Topic
	if subsResName == nil {
		// subscription was possibly never created
		return nil
	}

	subs := cli.Subscription(subsResName.Resource)

	cfg, err := subs.Config(ctx)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to Pub/Sub API. Ignoring: %s", toErrMsg(err))
		return nil
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed,
			fmt.Sprintf("Subscription %q not found, skipping deletion", subs))
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Failed to get Pub/Sub subscription %q: %s", subs, toErrMsg(err))
	}

	if !assertPubSubSubscriptionOwnership(src, cfg) {
		event.Warn(ctx, ReasonUnsubscribed, "Subscription %q is not owned by this source instance. "+
			"Skipping deletion", subs)
		return nil
	}

	// we don't pass cfg.Topic directly to error reporters below, because
	// (*pubsub.Topic).String() panics when Topic is nil
	var topicName string
	if subsTopic := cfg.Topic; subsTopic != nil {
		topicName = subsTopic.String()
	}

	err = subs.Delete(ctx)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to Pub/Sub API. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return failDeleteSubscriptionEvent(topicName, err)
	}

	event.Normal(ctx, ReasonUnsubscribed, "Unsubscribed from topic %q", topicName)

	return nil
}

// makeSubscriptionResourceName returns a Pub/Sub resource name for the given subscription.
func makeSubscriptionResourceName(proj, subsID string) *v1alpha1.GCloudResourceName {
	return &v1alpha1.GCloudResourceName{
		Project:    proj,
		Collection: pubsubCollectionSubs,
		Resource:   subsID,
	}
}

// pubsubResourceID returns a deterministic Pub/Sub resource ID matching the
// given source instance.
//
// Resource IDs aren't allowed to start with "goog" and can only contain a
// limited set of characters.
func pubsubResourceID(src commonv1alpha1.Reconcilable) string {
	return src.GetNamespace() + "." + src.GetName() + "~" + sources.GoogleCloudStorageSourceResource.String()
}

// assertPubSubTopicOwnership returns whether a Pub/Sub topic is owned by the
// given source.
func assertPubSubTopicOwnership(src *v1alpha1.GoogleCloudStorageSource, topicCfg pubsub.TopicConfig) bool {
	for k, v := range pubsubResourceLabels(src) {
		if topicCfg.Labels[k] != v {
			return false
		}
	}

	return true
}

// assertPubSubSubscriptionOwnership returns whether a Pub/Sub subscription is
// owned by the given source.
func assertPubSubSubscriptionOwnership(src *v1alpha1.GoogleCloudStorageSource, subsCfg pubsub.SubscriptionConfig) bool {
	for k, v := range pubsubResourceLabels(src) {
		if subsCfg.Labels[k] != v {
			return false
		}
	}

	return true
}

// pubsubResourceLabels returns a set of labels containing information from the
// given source instance to set on a Pub/Sub resource (topic or subscription).
//
// Labels accept lowercase characters, numbers, hyphens and underscores exclusively.
// Neither the key nor the value can exceed 63 characters.
func pubsubResourceLabels(src *v1alpha1.GoogleCloudStorageSource) map[string]string {
	return map[string]string{
		pubsubLabelOwnerResource:  strings.ReplaceAll(sources.GoogleCloudStorageSourceResource.String(), ".", "-"),
		pubsubLabelOwnerNamespace: src.Namespace,
		pubsubLabelOwnerName:      strings.ReplaceAll(src.Name, ".", "_"),
	}
}

// failGetTopicEvent returns a reconciler event which indicates that a Pub/Sub
// topic could not be retrieved from the Google Cloud API.
func failGetTopicEvent(topicName string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error getting topic %q: %s", topicName, toErrMsg(origErr))
}

// failCreateTopicEvent returns a reconciler event which indicates that a
// Pub/Sub topic could not be created via the Google Cloud API.
func failCreateTopicEvent(topicName string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error creating topic %q: %s", topicName, toErrMsg(origErr))
}

// failDeleteTopicEvent returns a reconciler event which indicates that
// a Pub/Sub subscription could not be deleted via the Google Cloud API.
func failDeleteTopicEvent(topicName string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
		"Error deleting topic %q: %s", topicName, toErrMsg(origErr))
}

// failGetSubscriptionEvent returns a reconciler event which indicates that a
// Pub/Sub subscription could not be retrieved from the Google Cloud API.
func failGetSubscriptionEvent(topicName string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error getting subscription for topic %q: %s", topicName, toErrMsg(origErr))
}

// failCreateSubscriptionEvent returns a reconciler event which indicates that a
// Pub/Sub subscription could not be created via the Google Cloud API.
func failCreateSubscriptionEvent(topicName string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error creating subscription for topic %q: %s", topicName, toErrMsg(origErr))
}

// failDeleteSubscriptionEvent returns a reconciler event which indicates that
// a Pub/Sub subscription could not be deleted via the Google Cloud API.
func failDeleteSubscriptionEvent(topicName string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
		"Error deleting subscription for topic %q: %s", topicName, toErrMsg(origErr))
}
