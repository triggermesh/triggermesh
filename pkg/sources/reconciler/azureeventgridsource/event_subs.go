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

package azureeventgridsource

import (
	"context"
	"fmt"
	"hash/crc32"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	azureeventgrid "github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/eventgrid"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
)

const crudTimeout = time.Second * 15

const (
	defaultMaxDeliveryAttempts = 30
	defaultEventTTL            = 1440
)

// ensureEventSubscription ensures an event subscription exists with the expected configuration.
// Required permissions:
//  - Microsoft.EventGrid/systemTopics/eventSubscriptions/read
//  - Microsoft.EventGrid/systemTopics/eventSubscriptions/write
//  - Microsoft.EventHub/namespaces/eventhubs/write
func ensureEventSubscription(ctx context.Context, cli eventgrid.EventSubscriptionsClient,
	sysTopicResID *v1alpha1.AzureResourceID, eventHubResID string) error {

	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureEventGridSource)

	status := &typedSrc.Status

	// read current event subscription

	rgName := sysTopicResID.ResourceGroup
	sysTopicName := sysTopicResID.ResourceName
	sysTopicResIDStr := sysTopicResID.String()
	subsName := eventSubscriptionResourceName(typedSrc)

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	currentEventSubs, err := cli.Get(restCtx, rgName, sysTopicName, subsName)
	switch {
	case isNotFound(err):
		// no-op
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to event subscription API: "+toErrMsg(err))
		return controller.NewPermanentError(failGetEventSubscriptionEvent(sysTopicResIDStr, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot look up event subscription: "+toErrMsg(err))
		// wrap any other error to fail the reconciliation
		return fmt.Errorf("%w", failGetEventSubscriptionEvent(sysTopicResIDStr, err))
	}

	subsExists := currentEventSubs.ID != nil

	// compare and create/update event subscription

	desiredEventSubs := newEventSubscription(eventHubResID, typedSrc.GetEventTypes())

	if equalEventSubscription(ctx, desiredEventSubs, currentEventSubs) {
		eventSubscriptionResID, err := parseResourceID(*currentEventSubs.ID)
		if err != nil {
			return fmt.Errorf("converting resource ID string to structured resource ID: %w", err)
		}

		status.EventSubscriptionID = eventSubscriptionResID
		status.MarkSubscribed()
		return nil
	}

	restCtx, cancel = context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	resultFuture, err := cli.CreateOrUpdate(restCtx, rgName, sysTopicName, subsName, desiredEventSubs)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to event subscription API: "+toErrMsg(err))
		return controller.NewPermanentError(failSubscribeEvent(sysTopicResIDStr, subsExists, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot subscribe to events: "+toErrMsg(err))
		return fmt.Errorf("%w", failSubscribeEvent(sysTopicResIDStr, subsExists, err))
	}

	if err := resultFuture.WaitForCompletionRef(ctx, cli.BaseClient()); err != nil {
		return fmt.Errorf("waiting for creation of event subscription %q: %w", subsName, err)
	}

	subsResult, err := resultFuture.Result(cli.ConcreteClient())
	if err != nil {
		return fmt.Errorf("reading created/updated event subscription %q: %w", subsName, err)
	}

	recordSubscribedEvent(ctx, *subsResult.ID, subsExists)

	eventSubscriptionResID, err := parseResourceID(*subsResult.ID)
	if err != nil {
		return fmt.Errorf("converting resource ID string to structured resource ID: %w", err)
	}

	status.EventSubscriptionID = eventSubscriptionResID
	status.MarkSubscribed()

	return nil
}

// newEventSubscription returns the desired state of the event subscription.
func newEventSubscription(eventHubResID string, eventTypes []string) azureeventgrid.EventSubscription {
	// Fields marked with a '*' below are attributes which would be
	// defaulted on creation by Azure if not explicitly set, but which we
	// set manually nevertheless in order to ease the comparison with the
	// current state in the main synchronization logic.

	return azureeventgrid.EventSubscription{
		EventSubscriptionProperties: &azureeventgrid.EventSubscriptionProperties{
			Destination: azureeventgrid.EventHubEventSubscriptionDestination{
				EndpointType: azureeventgrid.EndpointTypeEventHub,
				EventHubEventSubscriptionDestinationProperties: &azureeventgrid.EventHubEventSubscriptionDestinationProperties{
					ResourceID: to.StringPtr(eventHubResID),
				},
			},
			Filter: &azureeventgrid.EventSubscriptionFilter{
				IncludedEventTypes: to.StringSlicePtr(eventTypes),
				SubjectBeginsWith:  to.StringPtr(""), // *
				SubjectEndsWith:    to.StringPtr(""), // *
			},
			RetryPolicy: &azureeventgrid.RetryPolicy{
				MaxDeliveryAttempts:      to.Int32Ptr(defaultMaxDeliveryAttempts), // *
				EventTimeToLiveInMinutes: to.Int32Ptr(defaultEventTTL),            // *
			},
			EventDeliverySchema: azureeventgrid.EventDeliverySchemaCloudEventSchemaV10,
		},
	}
}

// ensureNoEventSubscription ensures the event subscription is removed.
// Required permissions:
//  - Microsoft.EventGrid/systemTopics/eventSubscriptions/delete
func ensureNoEventSubscription(ctx context.Context, cli eventgrid.EventSubscriptionsClient,
	sysTopic *azureeventgrid.SystemTopic) reconciler.Event {

	if skip.Skip(ctx) {
		return nil
	}

	if sysTopic == nil {
		event.Warn(ctx, ReasonUnsubscribed, "System topic not found, skipping finalization of event subscription")
		return nil
	}

	sysTopicResID, err := parseResourceID(*sysTopic.ID)
	if err != nil {
		return fmt.Errorf("converting resource ID string to structured resource ID: %w", err)
	}

	src := v1alpha1.ReconcilableFromContext(ctx)
	typedSrc := src.(*v1alpha1.AzureEventGridSource)

	rgName := sysTopicResID.ResourceGroup
	sysTopicName := sysTopicResID.ResourceName
	subsName := eventSubscriptionResourceName(typedSrc)

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	resultFuture, err := cli.Delete(restCtx, rgName, sysTopicName, subsName)
	switch {
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed, "Event subscription %q not found, skipping deletion", subsName)
		return nil
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to event subscription API. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return failUnsubscribeEvent(subsName, *sysTopic.ID, err)
	}

	if err := resultFuture.WaitForCompletionRef(ctx, cli.BaseClient()); err != nil {
		return fmt.Errorf("waiting for deletion of event subscription %q: %w", subsName, err)
	}

	event.Normal(ctx, ReasonUnsubscribed, "Deleted event subscription %q from system topic %q", subsName, *sysTopic.ID)

	return nil
}

// eventSubscriptionResourceName returns a deterministic name for an Event Grid
// event subscription associated with the given source instance.
// The Event Subscription name must be 3-64 characters in length and can only
// contain a-z, A-Z, 0-9, and "-".
func eventSubscriptionResourceName(src *v1alpha1.AzureEventGridSource) string {
	nsNameChecksum := crc32.ChecksumIEEE([]byte(src.Namespace + "/" + src.Name))
	return "io-triggermesh-azureeventgridsources-" + strconv.FormatUint(uint64(nsNameChecksum), 10)
}

// recordSubscribedEvent records a Kubernetes API event which indicates that an
// event subscription was either created or updated.
func recordSubscribedEvent(ctx context.Context, subsID string, isUpdate bool) {
	verb := "Created"
	if isUpdate {
		verb = "Updated"
	}

	event.Normal(ctx, ReasonSubscribed, "%s event subscription %q", verb, subsID)
}

// failGetEventSubscriptionEvent returns a reconciler event which indicates
// that an event subscription for the given system topic could not be retrieved
// from the Azure API.
func failGetEventSubscriptionEvent(sysTopic string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error getting event subscription from system topic %q: %s", sysTopic, toErrMsg(origErr))
}

// failSubscribeEvent returns a reconciler event which indicates that an event
// subscription for the given system topic could not be created or updated via
// the Azure API.
func failSubscribeEvent(sysTopic string, isUpdate bool, origErr error) reconciler.Event {
	verb := "creating"
	if isUpdate {
		verb = "updating"
	}

	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error %s event subscription in system topic %q: %s", verb, sysTopic, toErrMsg(origErr))
}

// failUnsubscribeEvent returns a reconciler event which indicates that an
// event subscription could not be deleted via the Azure API.
func failUnsubscribeEvent(subs, sysTopic string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error deleting event subscription %q from system topic %q: %s", subs, sysTopic, toErrMsg(origErr))
}
