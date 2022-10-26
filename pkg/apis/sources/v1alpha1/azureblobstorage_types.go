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

// AzureBlobStorageSource is the Schema for the event source.
type AzureBlobStorageSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureBlobStorageSourceSpec   `json:"spec,omitempty"`
	Status AzureBlobStorageSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AzureBlobStorageSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*AzureBlobStorageSource)(nil)
	_ v1alpha1.EventSource         = (*AzureBlobStorageSource)(nil)
	_ v1alpha1.EventSender         = (*AzureBlobStorageSource)(nil)
)

// AzureBlobStorageSourceSpec defines the desired state of the event source.
type AzureBlobStorageSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Resource ID of the Storage Account to receive events for.
	//
	// Format: /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{storageAccountName}
	//
	// Besides the Storage Account name itself, the resource ID contains
	// the subscription ID and resource group name which all together
	// uniquely identify the Storage Account within Azure.
	StorageAccountID AzureResourceID `json:"storageAccountID"`

	// Types of events to subscribe to.
	//
	// The list of available event types can be found at
	// https://docs.microsoft.com/en-us/azure/event-grid/event-schema-blob-storage
	//
	// When this attribute is not set, the source automatically subscribes
	// to the following event types:
	// - Microsoft.Storage.BlobCreated
	// - Microsoft.Storage.BlobDeleted
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

// AzureBlobStorageSourceStatus defines the observed state of the event source.
type AzureBlobStorageSourceStatus struct {
	v1alpha1.Status `json:",inline"`

	// Resource ID of the Event Hubs instance that is currently receiving
	// events from the Azure Event Grid subscription.
	EventHubID *AzureResourceID `json:"eventHubID,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureBlobStorageSourceList contains a list of event sources.
type AzureBlobStorageSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureBlobStorageSource `json:"items"`
}
