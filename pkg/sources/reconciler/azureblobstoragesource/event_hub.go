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
	"encoding/json"
	"fmt"
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
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/storage"
)

// We don't know the pricing tier of the Event Hubs namespace, so we default to
// the maximum of the "Basic" tier.
const (
	defaultMessageRetentionDays = 1
	defaultPartitionCount       = 4
)

const resourceTypeEventHubs = "eventhubs"

// EnsureEventHub ensures the existence of an Event Hub for sending storage events.
// Required permissions:
//   - Microsoft.EventHub/namespaces/eventhubs/read
//   - Microsoft.EventHub/namespaces/eventhubs/write
func EnsureEventHub(ctx context.Context, cli storage.EventHubsClient) (string /*resource ID*/, error) {
	if skip.Skip(ctx) {
		return "", nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.AzureBlobStorageSource)
	status := &src.Status

	if userProvidedHub := src.Spec.Endpoint.EventHubs.HubName; userProvidedHub != nil {
		eventHubID := makeEventHubID(&src.Spec.Endpoint.EventHubs.NamespaceID, *userProvidedHub)
		status.EventHubID = eventHubID
		return eventHubID.String(), nil
	}

	stAccName := src.Spec.StorageAccountID.ResourceName

	// the naming rule for Storage Accounts is more restrictive than the
	// one for Event Hubs, so this should hopefully always be valid, on top
	// of being deterministic
	eventHubName := stAccName
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
			return "", controller.NewPermanentError(failCreateEventHubEvent(stAccName, err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot create Event Hub: "+toErrMsg(err))
			return "", fmt.Errorf("%w", failCreateEventHubEvent(stAccName, err))
		}

		event.Normal(ctx, ReasonEventHubCreated, "Created Event Hub %q for storage account %q",
			*res.Name, stAccName)

	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Access denied to Event Hubs API: "+toErrMsg(err))
		return "", controller.NewPermanentError(failGetEventHubEvent(stAccName, err))

	case err != nil:
		status.MarkNotSubscribed(v1alpha1.AzureReasonAPIError, "Cannot look up Event Hub: "+toErrMsg(err))
		return "", fmt.Errorf("%w", failGetEventHubEvent(stAccName, err))
	}

	eventHubResID, err := parseEventHubResID(*res.ID)
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

// EnsureNoEventHub ensures that the Event Hub created for sending storage
// events is deleted.
// Required permissions:
//   - Microsoft.EventHub/namespaces/eventhubs/delete
func EnsureNoEventHub(ctx context.Context, cli storage.EventHubsClient) error {
	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.AzureBlobStorageSource)

	if userProvidedHub := src.Spec.Endpoint.EventHubs.HubName; userProvidedHub != nil {
		// do not delete Event Hubs managed by the user
		return nil
	}

	stAccName := src.Spec.StorageAccountID.ResourceName

	eventHubName := stAccName
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
		return failDeleteEventHubEvent(stAccName, err)
	}

	event.Normal(ctx, ReasonEventHubDeleted, "Deleted Event Hub %q for storage account %q",
		eventHubName, stAccName)

	return nil
}

// parseEventHubResID parses the given Event Hub resource ID string to a
// structured resource ID.
func parseEventHubResID(resIDStr string) (*v1alpha1.AzureResourceID, error) {
	resID := &v1alpha1.AzureResourceID{}

	err := json.Unmarshal([]byte(strconv.Quote(resIDStr)), resID)
	if err != nil {
		return nil, fmt.Errorf("deserializing resource ID string: %w", err)
	}

	return resID, nil
}

// failGetEventHubEvent returns a reconciler event which indicates that an
// Event Hub for the given storage account could not be retrieved from the
// Azure API.
func failGetEventHubEvent(stAcc string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedEventHub,
		"Error getting Event Hub for storage account %q: %s", stAcc, toErrMsg(origErr))
}

// failCreateEventHubEvent returns a reconciler event which indicates that an
// Event Hub could not be created via the Azure API.
func failCreateEventHubEvent(stAcc string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedEventHub,
		"Error creating Event Hub for storage account %q: %s", stAcc, toErrMsg(origErr))
}

// failDeleteEventHubEvent returns a reconciler event which indicates that an
// Event Hub could not be deleted via the Azure API.
func failDeleteEventHubEvent(stAcc string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedEventHub,
		"Error deleting Event Hub for storage account %q: %s", stAcc, toErrMsg(origErr))
}
