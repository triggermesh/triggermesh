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

package azureblobstoragesource

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/storage"
)

const crudTimeout = time.Second * 15

const (
	defaultMaxDeliveryAttempts = 30
	defaultEventTTL            = 1440
)

// EnsureEventSubscription ensures an event subscription exists with the expected configuration.
// Required permissions:
//   - Microsoft.EventGrid/eventSubscriptions/read
//   - Microsoft.EventGrid/eventSubscriptions/write
//   - Microsoft.EventHub/namespaces/eventhubs/write
func EnsureEventSubscription(ctx context.Context, cli storage.EventSubscriptionsClient, eventHubResID string) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureBlobStorageSource)

	status := &typedSrc.Status

	// read current event subscription

	stAccID := typedSrc.Spec.StorageAccountID.String()
	stAccName := typedSrc.Spec.StorageAccountID.ResourceName
	subsName := subscriptionName(typedSrc)

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	currentEventSubs, err := cli.Get(restCtx, stAccID, subsName)
	switch {
	case isNotFound(err):
		// no-op
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to event subscription API: "+toErrMsg(err))
		return controller.NewPermanentError(failGetEventSubscriptionEvent(stAccName, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot look up event subscription: "+toErrMsg(err))
		return fmt.Errorf("%w", failGetEventSubscriptionEvent(stAccName, err))
	}

	subsExists := currentEventSubs.ID != nil

	// compare and create/update event subscription

	desiredEventSubs := newEventSubscription(eventHubResID, typedSrc.GetEventTypes())

	if equalEventSubscription(ctx, desiredEventSubs, currentEventSubs) {
		status.MarkSubscribed()
		return nil
	}

	restCtx, cancel = context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	_, err = cli.CreateOrUpdate(restCtx, stAccID, subsName, desiredEventSubs)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to event subscription API: "+toErrMsg(err))
		return controller.NewPermanentError(failSubscribeEvent(stAccName, subsExists, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot subscribe to storage events: "+toErrMsg(err))
		return fmt.Errorf("%w", failSubscribeEvent(stAccName, subsExists, err))
	}

	recordSubscribedEvent(ctx, subsName, stAccName, subsExists)

	status.MarkSubscribed()

	return nil
}

// newEventSubscription returns the desired state of the event subscription for
// the given source.
func newEventSubscription(eventHubResID string, eventTypes []string) eventgrid.EventSubscription {
	// Fields marked with a '*' below are attributes which would be
	// defaulted on creation by Azure if not explicitly set, but which we
	// set manually nevertheless in order to ease the comparison with the
	// current state in the main synchronization logic.

	return eventgrid.EventSubscription{
		EventSubscriptionProperties: &eventgrid.EventSubscriptionProperties{
			Destination: eventgrid.EventHubEventSubscriptionDestination{
				EndpointType: eventgrid.EndpointTypeEventHub,
				EventHubEventSubscriptionDestinationProperties: &eventgrid.EventHubEventSubscriptionDestinationProperties{
					ResourceID: to.Ptr(eventHubResID),
				},
			},
			Filter: &eventgrid.EventSubscriptionFilter{
				IncludedEventTypes: to.Ptr(eventTypes),
				SubjectBeginsWith:  to.Ptr(""), // *
				SubjectEndsWith:    to.Ptr(""), // *
			},
			RetryPolicy: &eventgrid.RetryPolicy{
				MaxDeliveryAttempts:      to.Ptr[int32](defaultMaxDeliveryAttempts), // *
				EventTimeToLiveInMinutes: to.Ptr[int32](defaultEventTTL),            // *
			},
			EventDeliverySchema: eventgrid.EventDeliverySchemaCloudEventSchemaV10,
		},
	}
}

// EnsureNoEventSubscription ensures the event subscription is removed.
// Required permissions:
//   - Microsoft.EventGrid/eventSubscriptions/delete
func EnsureNoEventSubscription(ctx context.Context, cli storage.EventSubscriptionsClient) reconciler.Event {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureBlobStorageSource)

	stAccID := typedSrc.Spec.StorageAccountID.String()
	stAccName := typedSrc.Spec.StorageAccountID.ResourceName
	subsName := subscriptionName(typedSrc)

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	_, err := cli.Delete(restCtx, stAccID, subsName)
	switch {
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed, "Event subscription not found, skipping deletion")
		return nil
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to event subscription API. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return failUnsubscribeEvent(stAccName, err)
	}

	event.Normal(ctx, ReasonUnsubscribed, "Deleted event subscription %q for storage account %q",
		subsName, stAccName)

	return nil
}

// equalEventSubscription asserts the equality of two EventSubscriptions.
func equalEventSubscription(ctx context.Context, x, y eventgrid.EventSubscription) bool {
	cmpFn := cmp.Equal
	if logger := logging.FromContext(ctx); logger.Desugar().Core().Enabled(zapcore.DebugLevel) {
		cmpFn = diffLoggingCmp(logger)
	}
	return cmpFn(x.EventSubscriptionProperties, y.EventSubscriptionProperties,
		cmpopts.IgnoreFields(eventgrid.EventSubscriptionProperties{},
			// read-only fields are excluded
			"Topic",
			"ProvisioningState",
		),
		cmpopts.SortSlices(lessStrings),
	)
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

// subscriptionName returns a predictable name for an Event Grid event
// subscription associated with the given source instance.
// The Event Subscription name must be 3-64 characters in length and can only
// contain a-z, A-Z, 0-9, and "-".
func subscriptionName(src *v1alpha1.AzureBlobStorageSource) string {
	nsNameChecksum := crc32.ChecksumIEEE([]byte(src.Namespace + "/" + src.Name))
	return "io-triggermesh-azureblobstoragesource-" + strconv.FormatUint(uint64(nsNameChecksum), 10)
}

// cmpFunc can compare the equality of two interfaces. The function signature
// is the same as cmp.Equal.
type cmpFunc func(x, y interface{}, opts ...cmp.Option) bool

// diffLoggingCmp compares the equality of two interfaces and logs the diff at
// the Debug level.
func diffLoggingCmp(logger *zap.SugaredLogger) cmpFunc {
	return func(x, y interface{}, opts ...cmp.Option) bool {
		if diff := cmp.Diff(x, y, opts...); diff != "" {
			logger.Debug("Event subscriptions differ (-want, +got)\n" + diff)
			return false
		}
		return true
	}
}

// lessStrings reports whether the string element with index i should sort
// before the string element with index j.
func lessStrings(i, j string) bool {
	return i < j
}

// recordSubscribedEvent records a Kubernetes API event which indicates that an
// event subscription was either created or updated.
func recordSubscribedEvent(ctx context.Context, subsName, stAccName string, isUpdate bool) {
	verb := "Created"
	if isUpdate {
		verb = "Updated"
	}

	event.Normal(ctx, ReasonSubscribed, "%s event subscription %q for storage account %q",
		verb, subsName, stAccName)
}

// failGetEventSubscriptionEvent returns a reconciler event which indicates
// that an event subscription for the given storage account could not be
// retrieved from the Azure API.
func failGetEventSubscriptionEvent(stAcc string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error getting event subscription for storage account %q: %s", stAcc, toErrMsg(origErr))
}

// failSubscribeEvent returns a reconciler event which indicates that an event
// subscription for the given storage account could not be created or updated
// via the Azure API.
func failSubscribeEvent(stAcc string, isUpdate bool, origErr error) reconciler.Event {
	verb := "creating"
	if isUpdate {
		verb = "updating"
	}

	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error %s event subscription for storage account %q: %s", verb, stAcc, toErrMsg(origErr))
}

// failUnsubscribeEvent returns a reconciler event which indicates that an
// event subscription for the given storage account could not be deleted via
// the Azure API.
func failUnsubscribeEvent(stAcc string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error deleting event subscription for storage account %q: %s", stAcc, toErrMsg(origErr))
}
