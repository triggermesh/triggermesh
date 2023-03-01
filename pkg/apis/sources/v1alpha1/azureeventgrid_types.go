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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureEventGridSource is the Schema for the event source.
type AzureEventGridSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureEventGridSourceSpec   `json:"spec,omitempty"`
	Status AzureEventGridSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AzureEventGridSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*AzureEventGridSource)(nil)
	_ v1alpha1.EventSource         = (*AzureEventGridSource)(nil)
	_ v1alpha1.EventSender         = (*AzureEventGridSource)(nil)
)

// AzureEventGridSourceSpec defines the desired state of the event source.
type AzureEventGridSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// The resource ID the event subscription applies to.
	//
	// Can be
	// - an Azure subscription:
	//   /subscriptions/{subscriptionId}
	// - a resource group:
	//   /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}
	// - a top-level resource from a resource provider (including Event Grid topic):
	//   /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}
	Scope AzureResourceID `json:"scope"`

	// Types of events to subscribe to.
	//
	// If not specified, Azure automatically selects all available event types for the provided Scope.
	//
	// For a list of all available event types, please refer to the list of
	// Azure services that support system topics at
	// https://docs.microsoft.com/en-us/azure/event-grid/system-topics
	//
	// +optional
	EventTypes []string `json:"eventTypes,omitempty"`

	// The intermediate destination of events subscribed via Event Grid,
	// before they are retrieved by this event source.
	Endpoint AzureEventGridSourceEndpoint `json:"endpoint"`

	// Authentication method to interact with the Azure REST API.
	// This event source only supports the ServicePrincipal authentication.
	// If it not present, it will try to use Azure AKS Managed Identity
	Auth AzureAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// AzureEventGridSourceEndpoint contains possible intermediate destinations for events.
type AzureEventGridSourceEndpoint struct {
	EventHubs AzureEventGridSourceDestinationEventHubs `json:"eventHubs"`
}

// AzureEventGridSourceDestinationEventHubs contains properties of an Event
// Hubs namespace to use as intermediate destination for events.
type AzureEventGridSourceDestinationEventHubs struct {
	// Resource ID of the Event Hubs namespace.
	//
	// The expected format is
	//   /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}
	NamespaceID AzureResourceID `json:"namespaceID"`

	// Name of the Event Hubs instance within the selected namespace. If
	// omitted, an Event Hubs instance is created on behalf of the user.
	// +optional
	HubName *string `json:"hubName,omitempty"`

	// Name of the Event Hubs' Consumer Group that will be used by the source to read the event stream.
	ConsumerGroup *string `json:"consumerGroup,omitempty"`
}

// AzureEventGridSourceStatus defines the observed state of the event source.
type AzureEventGridSourceStatus struct {
	v1alpha1.Status `json:",inline"`

	// Resource ID of the Event Grid subscription that is currently
	// registered for the user-provided scope.
	EventSubscriptionID *AzureResourceID `json:"eventSubscriptionID,omitempty"`

	// Resource ID of the Event Hubs instance that is currently receiving
	// events from the Azure Event Grid subscription.
	EventHubID *AzureResourceID `json:"eventHubID,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureEventGridSourceList contains a list of event sources.
type AzureEventGridSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureEventGridSource `json:"items"`
}
