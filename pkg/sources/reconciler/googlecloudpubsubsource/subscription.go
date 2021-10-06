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

package googlecloudpubsubsource

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"cloud.google.com/go/pubsub"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
)

const (
	pubsubCollectionTopics = "topics"
	pubsubCollectionSubs   = "subscriptions"

	pubsubLabelOwnerResource  = "io-triggermesh_owner-resource"
	pubsubLabelOwnerNamespace = "io-triggermesh_owner-namespace"
	pubsubLabelOwnerName      = "io-triggermesh_owner-name"
)

// ensureSubscription ensures the existence of a Pub/Sub subscription for
// receiving messages from the desired topic.
// Required permissions:
// - pubsub.subscriptions.get
// - pubsub.subscriptions.create
func ensureSubscription(ctx context.Context, cli *pubsub.Client) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx).(*v1alpha1.GoogleCloudPubSubSource)
	status := &src.Status

	topicResName := src.Spec.Topic
	topicName := topicResName.String()

	// safeguard, in case the OpenAPI validation let a bad resource name through
	if topicResName.Collection != pubsubCollectionTopics {
		status.MarkNotSubscribed(v1alpha1.ReasonFailedSync, "Not a topic: "+topicName)
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"The provided resource name %q does not represent a Pub/Sub topic", topicName))
	}

	if userProvided := src.Spec.SubscriptionID; userProvided != nil {
		ok, err := belongsToTopic(ctx, cli, *userProvided, topicResName.Resource)
		switch {
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Access denied to Pub/Sub API: "+toErrMsg(err))
			return controller.NewPermanentError(failGetSubscriptionEvent(topicName, err))
		case isNotFound(err):
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError, "Provided subscription was not found")
			return controller.NewPermanentError(failGetSubscriptionEvent(topicName, err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Cannot look up subscription: "+toErrMsg(err))
			return fmt.Errorf("%w", failGetSubscriptionEvent(topicName, err))
		}

		if !ok {
			status.Subscription = nil

			status.MarkNotSubscribed(v1alpha1.ReasonFailedSync,
				"Subscription does not belong to provided topic "+strconv.Quote(topicName))
			return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
				"The provided subscription %q does not belong to the topic %q", *userProvided, topicName))
		}

		// it is essential that we propagate the subscription's name
		// here, otherwise BuildAdapter() won't be able to configure
		// the Pub/Sub adapter properly
		status.Subscription = makeSubscriptionResourceName(topicResName.Project, *userProvided)

		return nil
	}

	subsID := subscriptionID(src)

	subs := cli.Subscription(subsID)
	exists, err := subs.Exists(ctx)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError, "Access denied to Pub/Sub API: "+toErrMsg(err))
		return controller.NewPermanentError(failGetSubscriptionEvent(topicName, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError, "Cannot look up subscription: "+toErrMsg(err))
		return fmt.Errorf("%w", failGetSubscriptionEvent(topicName, err))
	}

	if !exists {
		cfg := pubsub.SubscriptionConfig{
			Topic:  cli.Topic(src.Spec.Topic.Resource),
			Labels: subscriptionLabels(src),
		}

		_, err := cli.CreateSubscription(ctx, subsID, cfg)
		switch {
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError, "Access denied to Pub/Sub API: "+toErrMsg(err))
			return controller.NewPermanentError(failCreateSubscriptionEvent(topicName, err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError, "Cannot subscribe to topic: "+toErrMsg(err))
			return fmt.Errorf("%w", failCreateSubscriptionEvent(topicName, err))
		}
	}

	// it is essential that we propagate the subscription's name
	// here, otherwise BuildAdapter() won't be able to configure
	// the Pub/Sub adapter properly
	status.Subscription = makeSubscriptionResourceName(topicResName.Project, subsID)

	if !status.GetCondition(v1alpha1.GoogleCloudPubSubConditionSubscribed).IsTrue() {
		event.Normal(ctx, ReasonSubscribed, "Subscribed to topic %q", topicName)
	}

	status.MarkSubscribed()

	return nil
}

// ensureNoSubscription ensures that the Pub/Sub subscription created for
// receiving messages from the desired topic is deleted.
// Required permissions:
// - pubsub.subscriptions.get
// - pubsub.subscriptions.delete
func ensureNoSubscription(ctx context.Context, cli *pubsub.Client) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx).(*v1alpha1.GoogleCloudPubSubSource)

	if src.Spec.SubscriptionID != nil {
		// do not delete subscriptions managed by the user
		return nil
	}

	subsID := subscriptionID(src)

	owns, err := assertOwnership(ctx, cli, src, subsID)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe, "Access denied to Pub/Sub API. Ignoring: %s", toErrMsg(err))
		return nil
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed, fmt.Sprintf("Subscription %q not found, skipping deletion", subsID))
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Failed to verify owner of Pub/Sub subscription %q: %s", subsID, toErrMsg(err))
	}

	if !owns {
		event.Warn(ctx, ReasonUnsubscribed, "Subscription %q is not owned by this source instance. "+
			"Skipping deletion", subsID)
		return nil
	}

	topicName := src.Spec.Topic.String()

	err = cli.Subscription(subsID).Delete(ctx)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe, "Access denied to Pub/Sub API. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return failDeleteSubscriptionEvent(topicName, err)
	}

	return reconciler.NewEvent(corev1.EventTypeNormal, ReasonUnsubscribed, "Unsubscribed from topic %q", topicName)
}

// belongsToTopic ensures that a Pub/Sub subscription belongs to the expected topic.
func belongsToTopic(ctx context.Context, cli *pubsub.Client, subsID, topicID string) (bool, error) {
	subs := cli.Subscription(subsID)
	cfg, err := subs.Config(ctx)
	if err != nil {
		return false, err
	}

	return cfg.Topic.ID() == topicID, nil
}

// makeSubscriptionResourceName returns a Pub/Sub resource name for the given subscription.
func makeSubscriptionResourceName(proj, subsID string) *v1alpha1.GCloudResourceName {
	return &v1alpha1.GCloudResourceName{
		Project:    proj,
		Collection: pubsubCollectionSubs,
		Resource:   subsID,
	}
}

// subscriptionID returns a deterministic Pub/Sub subscription ID matching the
// given source instance.
//
// Resource IDs aren't allowed to start with "goog" and can only contain a
// limited set of characters.
func subscriptionID(src v1alpha1.EventSource) string {
	return src.GetNamespace() + "." + src.GetName() + "~" + sources.GoogleCloudPubSubSourceResource.String()
}

// assertOwnership returns whether a Pub/Sub subscription is owned by the given source.
func assertOwnership(ctx context.Context, cli *pubsub.Client,
	src *v1alpha1.GoogleCloudPubSubSource, subsID string) (bool, error) {

	subs := cli.Subscription(subsID)
	cfg, err := subs.Config(ctx)
	if err != nil {
		return false, err
	}

	for k, v := range subscriptionLabels(src) {
		if cfg.Labels[k] != v {
			return false, nil
		}
	}

	return true, nil
}

// subscriptionLabels returns a set of labels containing information from the
// given source instance to set on a Pub/Sub subscription.
//
// Labels accept lowercase characters, numbers, hyphens and underscores exclusively.
// Neither the key nor the value can exceed 63 characters.
func subscriptionLabels(src *v1alpha1.GoogleCloudPubSubSource) map[string]string {
	return map[string]string{
		pubsubLabelOwnerResource:  strings.ReplaceAll(sources.GoogleCloudPubSubSourceResource.String(), ".", "-"),
		pubsubLabelOwnerNamespace: src.Namespace,
		pubsubLabelOwnerName:      strings.ReplaceAll(src.Name, ".", "_"),
	}
}

// toErrMsg returns the given error as a string.
// If the error is a Google RPC status, the message contained in this status is returned.
func toErrMsg(err error) string {
	if s, ok := grpcstatus.FromError(err); ok {
		return s.Message()
	}

	return err.Error()
}

// isNotFound returns whether the given error indicates that some Google Cloud
// resource was not found.
func isNotFound(err error) bool {
	return grpcstatus.Code(err) == grpccodes.NotFound
}

// isDenied returns whether the given error indicates that a request to the
// Google Cloud API could not be authorized.
// This category of issues is unrecoverable without user intervention.
func isDenied(err error) bool {
	return grpcstatus.Code(err) == grpccodes.PermissionDenied
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
