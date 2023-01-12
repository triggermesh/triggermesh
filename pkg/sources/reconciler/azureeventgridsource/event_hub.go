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

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/eventgrid"
)

// We don't know the pricing tier of the Event Hubs namespace, so we default to
// the maximum of the "Basic" tier.
const (
	defaultMessageRetentionDays = 1
	defaultPartitionCount       = 4
)

const resourceTypeEventHubs = "eventhubs"

// EnsureEventHub ensures the existence of an Event Hub for sending events.
// Required permissions:
//   - Microsoft.EventHub/namespaces/eventhubs/read
//   - Microsoft.EventHub/namespaces/eventhubs/write
func EnsureEventHub(ctx context.Context, cli eventgrid.EventHubsClient) (string /*resource ID*/, error) {
	if skip.Skip(ctx) {
		return "", nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.AzureEventGridSource)
	status := &src.Status

	if userProvidedHub := src.Spec.Endpoint.EventHubs.HubName; userProvidedHub != nil {
		eventHubID := makeEventHubID(&src.Spec.Endpoint.EventHubs.NamespaceID, *userProvidedHub)
		status.EventHubID = eventHubID
		return eventHubID.String(), nil
	}

	scope := src.Spec.Scope.String()

	eventHubName := makeEventHubName(src)
	resourceGroup := src.Spec.Endpoint.EventHubs.NamespaceID.ResourceGroup
	namespace := src.Spec.Endpoint.EventHubs.NamespaceID.ResourceName

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	res, err := cli.Get(restCtx, resourceGroup, namespace, eventHubName)
	switch {
	case isNotFound(err):
		eventHubProps := eventhub.Model{
			Properties: &eventhub.Properties{
				PartitionCount:         to.Ptr[int64](defaultPartitionCount),
				MessageRetentionInDays: to.Ptr[int64](defaultMessageRetentionDays),
			},
		}

		restCtx, cancel = context.WithTimeout(ctx, crudTimeout)
		defer cancel()

		res, err = cli.CreateOrUpdate(restCtx, resourceGroup, namespace, eventHubName, eventHubProps)
		switch {
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to Event Hubs API: "+toErrMsg(err))
			return "", controller.NewPermanentError(failCreateEventHubEvent(scope, err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot create Event Hub: "+toErrMsg(err))
			return "", fmt.Errorf("%w", failCreateEventHubEvent(scope, err))
		}

		event.Normal(ctx, ReasonEventHubCreated, "Created Event Hub %q for Azure resource %q",
			*res.Name, scope)

	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to Event Hubs API: "+toErrMsg(err))
		return "", controller.NewPermanentError(failGetEventHubEvent(scope, err))

	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot look up Event Hub: "+toErrMsg(err))
		return "", fmt.Errorf("%w", failGetEventHubEvent(scope, err))
	}

	eventHubResID, err := parseResourceID(*res.ID)
	if err != nil {
		return "", fmt.Errorf("converting resource ID string to structured resource ID: %w", err)
	}

	// it is essential that we propagate the Event Hub resource ID here,
	// otherwise BuildAdapter() won't be able to configure the Event Hubs
	// adapter properly
	status.EventHubID = eventHubResID

	return *res.ID, nil
}

// makeEventHubID returns the Resource ID of an Event Hubs instance based on
// the given Event Hubs namespace and Hub name.
func makeEventHubID(namespaceID *v1alpha1.AzureResourceID, hubName string) *v1alpha1.AzureResourceID {
	hubID := *namespaceID
	hubID.Namespace = namespaceID.ResourceName
	hubID.ResourceType = resourceTypeEventHubs
	hubID.ResourceName = hubName
	return &hubID
}

// EnsureNoEventHub ensures that the Event Hub created for sending events
// is deleted.
// Required permissions:
//   - Microsoft.EventHub/namespaces/eventhubs/delete
func EnsureNoEventHub(ctx context.Context, cli eventgrid.EventHubsClient) error {
	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.AzureEventGridSource)

	if userProvidedHub := src.Spec.Endpoint.EventHubs.HubName; userProvidedHub != nil {
		// do not delete Event Hubs managed by the user
		return nil
	}

	scope := src.Spec.Scope.String()

	eventHubName := makeEventHubName(src)
	resourceGroup := src.Spec.Endpoint.EventHubs.NamespaceID.ResourceGroup
	namespace := src.Spec.Endpoint.EventHubs.NamespaceID.ResourceName

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	_, err := cli.Delete(restCtx, resourceGroup, namespace, eventHubName)
	switch {
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed, "Event Hub not found, skipping deletion")
		return nil
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedEventHub,
			"Access denied to Event Hubs API. Ignoring: %s", toErrMsg(err))
		return nil
	case err != nil:
		return failDeleteEventHubEvent(scope, err)
	}

	event.Normal(ctx, ReasonEventHubDeleted, "Deleted Event Hub %q for Azure resource %q",
		eventHubName, scope)

	return nil
}

// makeEventHubName returns a deterministic name for an Event Hubs instance.
//
// The generated name must match the regexp /[a-zA-Z0-9][\w.-]{0,49}/, which doesn't give us a lot of characters for
// indicating what component owns the Event Hub. Therefore, we compute the CRC32 checksum of the source's
// name/namespace (8 characters) and make it part of the name.
func makeEventHubName(src *v1alpha1.AzureEventGridSource) string {
	nsNameChecksum := crc32.ChecksumIEEE([]byte(src.Namespace + "/" + src.Name))
	return "io.triggermesh.azureeventgridsources-" + strconv.FormatUint(uint64(nsNameChecksum), 10)
}

// failGetEventHubEvent returns a reconciler event which indicates that an
// Event Hub for the given Azure resource could not be retrieved from the
// Azure API.
func failGetEventHubEvent(resource string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedEventHub,
		"Error getting Event Hub for Azure resource %q: %s", resource, toErrMsg(origErr))
}

// failCreateEventHubEvent returns a reconciler event which indicates that an
// Event Hub could not be created via the Azure API.
func failCreateEventHubEvent(resource string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedEventHub,
		"Error creating Event Hub for Azure resource %q: %s", resource, toErrMsg(origErr))
}

// failDeleteEventHubEvent returns a reconciler event which indicates that an
// Event Hub could not be deleted via the Azure API.
func failDeleteEventHubEvent(resource string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedEventHub,
		"Error deleting Event Hub for Azure resource %q: %s", resource, toErrMsg(origErr))
}
