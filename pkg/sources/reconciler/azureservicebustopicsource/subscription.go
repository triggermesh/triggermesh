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

package azureservicebustopicsource

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	servicebussdk "github.com/Azure/azure-service-bus-go"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/servicebus"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
)

const resourceTypeSubscriptions = "subscriptions"

// ensureSubscription ensures a Subscription exists with the expected configuration.
// Required permissions:
//  - Microsoft.ServiceBus/namespaces/topics/subscriptions/read
//  - Microsoft.ServiceBus/namespaces/topics/subscriptions/write
func ensureSubscription(ctx context.Context, cli *servicebus.Namespace) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureServiceBusTopicSource)

	status := &typedSrc.Status

	topicID := &typedSrc.Spec.TopicID

	subsMngr, err := cli.NewSubscriptionManager(topicID.ResourceName)
	if err != nil {
		return fmt.Errorf("obtaining Subscription manager for topic %q: %w", topicID, err)
	}

	subsName := subscriptionName(src)

	subs, err := subsMngr.Get(ctx, subsName)
	switch {
	case isNotFound(err):
		subs, err = subsMngr.Put(ctx, subsName)
		switch {
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to Subscription API: "+toErrMsg(err))
			return controller.NewPermanentError(failCreateSubscriptionEvent(topicID, err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError, "Cannot subscribe to Topic: "+toErrMsg(err))
			return fmt.Errorf("%w", failCreateSubscriptionEvent(topicID, err))
		}

	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to Subscription API: "+toErrMsg(err))
		return controller.NewPermanentError(failGetSubscriptionEvent(topicID, err))

	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot look up Subscription: "+toErrMsg(err))
		return fmt.Errorf("%w", failGetSubscriptionEvent(topicID, err))
	}

	if !status.GetCondition(v1alpha1.AzureServiceBusTopicConditionSubscribed).IsTrue() {
		event.Normal(ctx, ReasonSubscribed, "Subscribed to Topic %q", topicID)
	}

	subscriptionResID := makeSubscriptionResourceID(topicID, subs.Name)

	// it is essential that we propagate the subscription's resource ID
	// here, otherwise BuildAdapter() won't be able to configure the
	// Service Bus adapter properly
	status.SubscriptionID = subscriptionResID
	status.MarkSubscribed()

	return nil
}

// ensureNoSubscription ensures the Subscription is removed.
// Required permissions:
//  - Microsoft.ServiceBus/namespaces/topics/subscriptions/delete
func ensureNoSubscription(ctx context.Context, cli *servicebus.Namespace) reconciler.Event {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureServiceBusTopicSource)

	topicID := &typedSrc.Spec.TopicID

	subsMngr, err := cli.NewSubscriptionManager(topicID.ResourceName)
	if err != nil {
		return fmt.Errorf("obtaining Subscription manager for topic %q: %w", topicID, err)
	}

	subsName := subscriptionName(src)

	err = subsMngr.Delete(ctx, subsName)
	switch {
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed, "Subscription not found, skipping deletion")
		return nil
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to Subscription API. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return failUnsubscribeEvent(topicID, err)
	}

	event.Normal(ctx, ReasonUnsubscribed, "Deleted Subscription %q for Topic %q",
		subsName, topicID)

	return nil
}

// makeSubscriptionResourceID returns a structured resource ID for a
// Subscription based on the resource ID of the parent topic.
func makeSubscriptionResourceID(topicID *v1alpha1.AzureResourceID, subsName string) *v1alpha1.AzureResourceID {
	subsID := *topicID
	subsID.SubResourceType = resourceTypeSubscriptions
	subsID.SubResourceName = subsName
	return &subsID
}

// toErrMsg returns the given error as a string.
// If the error is an Azure API error, the error message is sanitized while
// still preserving the concatenation of all nested levels of errors.
//
// Used to remove clutter from errors before writing them to status conditions.
func toErrMsg(err error) string {
	return recursErrMsg("", err)
}

// recursErrMsg concatenates the messages of deeply nested API errors recursively.
func recursErrMsg(errMsg string, err error) string {
	if errMsg != "" {
		errMsg += ": "
	}

	switch tErr := err.(type) {
	case autorest.DetailedError:
		return recursErrMsg(errMsg+tErr.Message, tErr.Original)
	case *azure.RequestError:
		if tErr.DetailedError.Original != nil {
			return recursErrMsg(errMsg+tErr.DetailedError.Message, tErr.DetailedError.Original)
		}
		if tErr.ServiceError != nil {
			return errMsg + tErr.ServiceError.Message
		}
	case adal.TokenRefreshError:
		// This type of error is returned when the OAuth authentication with Azure Active Directory fails, often
		// due to an invalid or expired secret.
		//
		// The associated message is typically opaque and contains elements that are unique to each request
		// (trace/correlation IDs, timestamps), which causes an infinite loop of reconciliation if propagated to
		// the object's status conditions.
		// Instead of resorting to over-engineered error parsing techniques to get around the verbosity of the
		// message, we simply return a short and generic error description.
		return errMsg + "failed to refresh token: the provided secret is either invalid or expired"
	}

	return errMsg + err.Error()
}

// isNotFound returns whether the given error indicates that some Azure
// resource was not found.
func isNotFound(err error) bool {
	return errors.Is(err, servicebussdk.ErrNotFound{})
}

// isDenied returns whether the given error indicates that a request to the
// Azure API could not be authorized.
// This category of issues is unrecoverable without user intervention.
func isDenied(err error) bool {
	// FIXME(antoineco): the SDK returns unstructured errors
	// request failed: 401 SubCode=40100: Unauthorized : Unauthorized access for 'DeleteSubscription' operation on endpoint 'sb://triggermesh-dev.servicebus.windows.net/triggermesh-dev/subscriptions/io.triggermesh.azureservicebussources-4197625275?api-version=2017-04'. Tracking Id: bbbccdfa-9277-407f-891a-18a59fa82c7c_G20
	return strings.Contains(err.Error(), "Unauthorized access")
}

// subscriptionName returns a deterministic Subscription name matching the
// given source instance.
//
// The generated name must match the regexp /[A-Za-z0-9][\w.-]{0,49}/, which
// doesn't give us a lot of characters for indicating what component owns the
// Subscription. Thereforce, we compute the CRC32 checksum of the source's
// name/namespace (8 characters) and make it part of the name.
func subscriptionName(src v1alpha1.EventSource) string {
	nsNameChecksum := crc32.ChecksumIEEE([]byte(src.GetNamespace() + "/" + src.GetName()))
	return "io.triggermesh.azureservicebussources-" + strconv.FormatUint(uint64(nsNameChecksum), 10)
}

// failGetSubscriptionEvent returns a reconciler event which indicates that a
// Subscription for the given Topic could not be retrieved from the Azure API.
func failGetSubscriptionEvent(topicID *v1alpha1.AzureResourceID, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error getting Subscription for Topic %q: %s", topicID, toErrMsg(origErr))
}

// failCreateSubscriptionEvent returns a reconciler event which indicates that
// a Subscription for the given Topic could not be created via the Azure API.
func failCreateSubscriptionEvent(topicID *v1alpha1.AzureResourceID, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error creating Subscription for Topic %q: %s", topicID, toErrMsg(origErr))
}

// failUnsubscribeEvent returns a reconciler event which indicates that a
// Subscription for the given Topic could not be deleted via the Azure API.
func failUnsubscribeEvent(topicID *v1alpha1.AzureResourceID, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error deleting Subscription for Topic %q: %s", topicID, toErrMsg(origErr))
}
