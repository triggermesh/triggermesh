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
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/servicebustopics"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
)

const crudTimeout = time.Second * 15

// ensureSubscription ensures a Subscription exists with the expected configuration.
// Required permissions:
//  - Microsoft.ServiceBus/namespaces/topics/subscriptions/read
//  - Microsoft.ServiceBus/namespaces/topics/subscriptions/write
func ensureSubscription(ctx context.Context, cli servicebustopics.SubscriptionsClient) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureServiceBusTopicSource)

	status := &typedSrc.Status

	// read current Subscription

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

	subsExists := currentSubs.ID != nil

	// compare and create/update Subscription

	desiredSubs := newSubscription()

	if equalSubscriptions(ctx, desiredSubs, currentSubs) {
		subscriptionResID, err := parseSubscriptionResID(*currentSubs.ID)
		if err != nil {
			return fmt.Errorf("converting resource ID string to structured resource ID: %w", err)
		}

		status.SubscriptionID = subscriptionResID
		status.MarkSubscribed()
		return nil
	}

	restCtx, cancel = context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	res, err := cli.CreateOrUpdate(restCtx, typedSrc.Spec.TopicID.ResourceGroup, typedSrc.Spec.TopicID.Namespace,
		typedSrc.Spec.TopicID.ResourceName, subsName, desiredSubs)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to Subscription API: "+toErrMsg(err))
		return controller.NewPermanentError(failSubscribeEvent(topic, subsExists, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot subscribe to events: "+toErrMsg(err))
		return fmt.Errorf("%w", failSubscribeEvent(topic, subsExists, err))
	}

	recordSubscribedEvent(ctx, subsName, topic, subsExists)

	subscriptionResID, err := parseSubscriptionResID(*res.ID)
	if err != nil {
		return fmt.Errorf("converting resource ID string to structured resource ID: %w", err)
	}

	status.SubscriptionID = subscriptionResID
	status.MarkSubscribed()

	return nil
}

// ensureNoSubscription ensures the Subscription is removed.
// Required permissions:
//  - Microsoft.ServiceBus/namespaces/topics/subscriptions/delete
func ensureNoSubscription(ctx context.Context, cli servicebustopics.SubscriptionsClient) reconciler.Event {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureServiceBusTopicSource)

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

// newSubscription returns the desired state of the Subscription for the given source.
// All values correspond to Azure's defaults.
func newSubscription() servicebus.SBSubscription {
	const maxDeliveryCount = 10

	// 1 minute in ISO 8601 duration format.
	// https://en.wikipedia.org/wiki/ISO_8601#Durations
	const oneMinuteISO8601 = "PT1M"
	// Max signed 64-bit integer in ISO 8601 duration format.
	// https://docs.microsoft.com/en-us/azure/service-bus-messaging/message-expiration
	const maxDurationISO8601 = "P10675199DT2H48M5.4775807S"

	return servicebus.SBSubscription{
		SBSubscriptionProperties: &servicebus.SBSubscriptionProperties{
			MaxDeliveryCount:                          to.Int32Ptr(maxDeliveryCount),
			LockDuration:                              to.StringPtr(oneMinuteISO8601),
			DefaultMessageTimeToLive:                  to.StringPtr(maxDurationISO8601),
			AutoDeleteOnIdle:                          to.StringPtr(maxDurationISO8601), // never
			RequiresSession:                           to.BoolPtr(false),
			EnableBatchedOperations:                   to.BoolPtr(true),
			DeadLetteringOnFilterEvaluationExceptions: to.BoolPtr(true),
			DeadLetteringOnMessageExpiration:          to.BoolPtr(false),
		},
	}
}

// equalSubscriptions asserts the equality of two SBSubscription instances.
func equalSubscriptions(ctx context.Context, desired, current servicebus.SBSubscription) bool {
	cmpFn := cmp.Equal
	if logger := logging.FromContext(ctx); logger.Desugar().Core().Enabled(zapcore.DebugLevel) {
		cmpFn = diffLoggingCmp(logger)
	}
	return cmpFn(desired.SBSubscriptionProperties, current.SBSubscriptionProperties,
		cmpopts.IgnoreFields(servicebus.SBSubscriptionProperties{},
			"MessageCount", "CreatedAt", "AccessedAt", "UpdatedAt", "CountDetails", "Status"),
	)
}

// cmpFunc can compare the equality of two interfaces. The function signature
// is the same as cmp.Equal.
type cmpFunc func(x, y interface{}, opts ...cmp.Option) bool

// diffLoggingCmp compares the equality of two interfaces and logs the diff at
// the Debug level.
func diffLoggingCmp(logger *zap.SugaredLogger) cmpFunc {
	return func(desired, current interface{}, opts ...cmp.Option) bool {
		if diff := cmp.Diff(desired, current, opts...); diff != "" {
			logger.Debug("Subscriptions differ (-desired, +current)\n" + diff)
			return false
		}
		return true
	}
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
		return errMsg + "Invalid client secret"
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
// Subscription. Thereforce, we compute the CRC32 checksum of the source's
// name/namespace (8 characters) and make it part of the name.
func subscriptionName(src v1alpha1.EventSource) string {
	nsNameChecksum := crc32.ChecksumIEEE([]byte(src.GetNamespace() + "/" + src.GetName()))
	return "io.triggermesh.azureservicebussources-" + strconv.FormatUint(uint64(nsNameChecksum), 10)
}

// recordSubscribedEvent records a Kubernetes API event which indicates that a
// Subscription was either created or updated.
func recordSubscribedEvent(ctx context.Context, subsName, topicID string, isUpdate bool) {
	verb := "Created"
	if isUpdate {
		verb = "Updated"
	}

	event.Normal(ctx, ReasonSubscribed, "%s Subscription %q for Topic %q",
		verb, subsName, topicID)
}

// failGetSubscriptionEvent returns a reconciler event which indicates that a
// Subscription for the given Topic could not be retrieved from the Azure API.
func failGetSubscriptionEvent(topicID string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error getting Subscription for Topic %q: %s", topicID, toErrMsg(origErr))
}

// failSubscribeEvent returns a reconciler event which indicates that a
// Subscription for the given Topic could not be created or updated via the
// Azure API.
func failSubscribeEvent(topicID string, isUpdate bool, origErr error) reconciler.Event {
	verb := "creating"
	if isUpdate {
		verb = "updating"
	}

	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error %s Subscription for Topic %q: %s", verb, topicID, toErrMsg(origErr))
}

// failUnsubscribeEvent returns a reconciler event which indicates that a
// Subscription for the given Topic could not be deleted via the Azure API.
func failUnsubscribeEvent(topicID string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error deleting Subscription for Topic %q: %s", topicID, toErrMsg(origErr))
}
