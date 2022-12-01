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

// AzureServiceBusTopicSource is the Schema for the event source.
type AzureServiceBusTopicSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureServiceBusTopicSourceSpec   `json:"spec,omitempty"`
	Status AzureServiceBusTopicSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AzureServiceBusTopicSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*AzureServiceBusTopicSource)(nil)
	_ v1alpha1.EventSource         = (*AzureServiceBusTopicSource)(nil)
	_ v1alpha1.EventSender         = (*AzureServiceBusTopicSource)(nil)
)

// AzureServiceBusTopicSourceSpec defines the desired state of the event source.
type AzureServiceBusTopicSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// The resource ID the Service Bus Topic to subscribe to.
	//
	// Expected format:
	// - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceBus/namespaces/{namespaceName}/topics/{topicName}
	TopicID AzureResourceID `json:"topicID"`

	// Authentication method to interact with the Azure REST API.
	// This event source only supports the ServicePrincipal authentication.
	// If it not present, it will try to use Azure AKS Managed Identity
	Auth AzureAuth `json:"auth"`

	// WebSocketsEnable
	// +optional
	WebSocketsEnable *bool `json:"webSocketsEnable,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// AzureServiceBusTopicSourceStatus defines the observed state of the event source.
type AzureServiceBusTopicSourceStatus struct {
	v1alpha1.Status `json:",inline"`

	// Resource ID of the Service Bus Subscription that is currently used
	// by the event source for consuming events from the configured Service
	// Bus Topic.
	SubscriptionID *AzureResourceID `json:"subscriptionID,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureServiceBusTopicSourceList contains a list of event sources.
type AzureServiceBusTopicSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureServiceBusTopicSource `json:"items"`
}
