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
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/storage"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
)

// We don't know the pricing tier of the Event Hubs namespace, so we default to
// the maximum of the "Basic" tier.
const (
	defaultMessageRetentionDays = 1
	defaultPartitionCount       = 4
)

// ensureEventHub ensures the existence of an Event Hub for sending storage events.
// Required permissions:
//  - Microsoft.EventHub/namespaces/eventhubs/read
//  - Microsoft.EventHub/namespaces/eventhubs/write
func ensureEventHub(ctx context.Context, cli storage.EventHubsClient) (string /*resource ID*/, error) {
	if skip.Skip(ctx) {
		return "", nil
	}

	src := v1alpha1.SourceFromContext(ctx).(*v1alpha1.AzureBlobStorageSource)
	status := &src.Status

	if userProvided := src.Spec.EventHubID; userProvided.EventHub != "" {
		status.EventHubID = &userProvided
		return userProvided.String(), nil
	}

	stAccName := src.Spec.StorageAccountID.StorageAccount

	// the naming rule for Storage Accounts is more restrictive than the
	// one for Event Hubs, so this should hopefully always be valid, on top
	// of being deterministic
	eventHubName := stAccName
	resourceGroup := src.Spec.EventHubID.ResourceGroup
	namespace := src.Spec.EventHubID.Namespace

	restCtx, cancel := context.WithTimeout(ctx, crudTimeout)
	defer cancel()

	res, err := cli.Get(restCtx, resourceGroup, namespace, eventHubName)
	switch {
	case isNotFound(err):
		eventHubProps := eventhub.Model{
			Properties: &eventhub.Properties{
				PartitionCount:         to.Int64Ptr(defaultPartitionCount),
				MessageRetentionInDays: to.Int64Ptr(defaultMessageRetentionDays),
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

// ensureNoEventHub ensures that the Event Hub created for sending storage
// events is deleted.
// Required permissions:
//  - Microsoft.EventHub/namespaces/eventhubs/delete
func ensureNoEventHub(ctx context.Context, cli storage.EventHubsClient) error {
	src := v1alpha1.SourceFromContext(ctx).(*v1alpha1.AzureBlobStorageSource)

	if userProvided := src.Spec.EventHubID; userProvided.EventHub != "" {
		// do not delete Event Hubs managed by the user
		return nil
	}

	stAccName := src.Spec.StorageAccountID.StorageAccount

	eventHubName := stAccName
	resourceGroup := src.Spec.EventHubID.ResourceGroup
	namespace := src.Spec.EventHubID.Namespace

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
func parseEventHubResID(resIDStr string) (*v1alpha1.EventHubResourceID, error) {
	resID := &v1alpha1.EventHubResourceID{}

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
