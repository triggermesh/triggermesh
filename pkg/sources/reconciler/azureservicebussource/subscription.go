/*
Copyright 2023 TriggerMesh Inc.

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

package azureservicebussource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/servicebustopics"
)

const crudTimeout = time.Second * 15

// EnsureSubscription ensures a Subscription exists with the expected configuration.
// Required permissions:
//   - Microsoft.ServiceBus/namespaces/topics/subscriptions/read
//   - Microsoft.ServiceBus/namespaces/topics/subscriptions/write
func EnsureSubscription(ctx context.Context, cli servicebustopics.SubscriptionsClient) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureServiceBusSource)

	status := &typedSrc.Status

	topic := typedSrc.Spec.TopicID.String()
	subsName := subscriptionName(src)

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	currentSubs, err := cli.Get(restCtx, typedSrc.Spec.TopicID.ResourceGroup, typedSrc.Spec.TopicID.Namespace,
		typedSrc.Spec.TopicID.ResourceName, subsName)
	switch {
	case isNotFound(err):
		// no-op
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to Subscription API: "+toErrMsg(err))
		return controller.NewPermanentError(failGetSubscriptionEvent(topic, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot look up Subscription: "+toErrMsg(err))
		return fmt.Errorf("%w", failGetSubscriptionEvent(topic, err))
	}

	if subsExists := currentSubs.ID != nil; !subsExists {
		// use Azure's defaults
		desiredSubs := servicebus.SBSubscription{}

		restCtx, cancel = context.WithTimeout(ctx, crudTimeout)
		defer cancel()

		currentSubs, err = cli.CreateOrUpdate(restCtx, typedSrc.Spec.TopicID.ResourceGroup, typedSrc.Spec.TopicID.Namespace,
			typedSrc.Spec.TopicID.ResourceName, subsName, desiredSubs)
		switch {
		// This call responds with NotFound if the topic doesn't exist.
		case isNotFound(err):
			// We use a generic error message in the object's status instead of the original error, because
			// these API errors tend to contain a confusing message ("incoming request is not recognized as
			// a namespace policy put request") and unique elements (timestamp and correlation ID) which we
			// don't want to cause inifinite loops of reconciliation.
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Topic does not exist")
			return controller.NewPermanentError(failSubscribeEvent(topic, err))
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to Subscription API: "+toErrMsg(err))
			return controller.NewPermanentError(failSubscribeEvent(topic, err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot subscribe to Topic: "+toErrMsg(err))
			return fmt.Errorf("%w", failSubscribeEvent(topic, err))
		}
	}

	if !status.GetCondition(v1alpha1.AzureServiceBusTopicConditionSubscribed).IsTrue() {
		event.Normal(ctx, ReasonSubscribed, "Created Subscription %q for Topic %q", subsName, topic)
	}

	// it is essential that we propagate the subscription's resource ID
	// here, otherwise BuildAdapter() won't be able to configure the
	// Service Bus adapter properly
	subscriptionResID, err := parseSubscriptionResID(*currentSubs.ID)
	if err != nil {
		return fmt.Errorf("converting resource ID string to structured resource ID: %w", err)
	}

	status.SubscriptionID = subscriptionResID
	status.MarkSubscribed()

	return nil
}

// EnsureNoSubscription ensures the Subscription is removed.
// Required permissions:
//   - Microsoft.ServiceBus/namespaces/topics/subscriptions/delete
func EnsureNoSubscription(ctx context.Context, cli servicebustopics.SubscriptionsClient) reconciler.Event {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureServiceBusSource)

	topic := typedSrc.Spec.TopicID.String()
	subsName := subscriptionName(src)

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	_, err := cli.Delete(restCtx, typedSrc.Spec.TopicID.ResourceGroup, typedSrc.Spec.TopicID.Namespace,
		typedSrc.Spec.TopicID.ResourceName, subsName)
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
		return failUnsubscribeEvent(topic, err)
	}

	event.Normal(ctx, ReasonUnsubscribed, "Deleted Subscription %q for Topic %q",
		subsName, topic)

	return nil
}

// parseSubscriptionResID parses the given Subscription resource ID string to a
// structured resource ID.
func parseSubscriptionResID(resIDStr string) (*v1alpha1.AzureResourceID, error) {
	resID := &v1alpha1.AzureResourceID{}

	err := json.Unmarshal([]byte(strconv.Quote(resIDStr)), resID)
	if err != nil {
		return nil, fmt.Errorf("deserializing resource ID string: %w", err)
	}

	return resID, nil
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
	if dErr := (autorest.DetailedError{}); errors.As(err, &dErr) {
		return dErr.StatusCode == http.StatusNotFound
	}
	return false
}

// isDenied returns whether the given error indicates that a request to the
// Azure API could not be authorized.
// This category of issues is unrecoverable without user intervention.
func isDenied(err error) bool {
	if dErr := (autorest.DetailedError{}); errors.As(err, &dErr) {
		if code, ok := dErr.StatusCode.(int); ok {
			return code == http.StatusUnauthorized || code == http.StatusForbidden
		}
	}

	return false
}

// subscriptionName returns a deterministic Subscription name matching the
// given source instance.
//
// The generated name must match the regexp /[A-Za-z0-9][\w.-]{0,49}/, which
// doesn't give us a lot of characters for indicating what component owns the
// Subscription. Therefore, we compute the CRC32 checksum of the source's
// name/namespace (8 characters) and make it part of the name.
func subscriptionName(src commonv1alpha1.Reconcilable) string {
	nsNameChecksum := crc32.ChecksumIEEE([]byte(src.GetNamespace() + "/" + src.GetName()))
	return "io.triggermesh.azureservicebussources-" + strconv.FormatUint(uint64(nsNameChecksum), 10)
}

// failGetSubscriptionEvent returns a reconciler event which indicates that a
// Subscription for the given Topic could not be retrieved from the Azure API.
func failGetSubscriptionEvent(topicID string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error getting Subscription for Topic %q: %s", topicID, toErrMsg(origErr))
}

// failSubscribeEvent returns a reconciler event which indicates that a
// Subscription for the given Topic could not be created via the Azure API.
func failSubscribeEvent(topicID string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error creating Subscription for Topic %q: %s", topicID, toErrMsg(origErr))
}

// failUnsubscribeEvent returns a reconciler event which indicates that a
// Subscription for the given Topic could not be deleted via the Azure API.
func failUnsubscribeEvent(topicID string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error deleting Subscription for Topic %q: %s", topicID, toErrMsg(origErr))
}
