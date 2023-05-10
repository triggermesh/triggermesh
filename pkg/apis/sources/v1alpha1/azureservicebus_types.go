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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureServiceBusSource is the Schema for the event source.
type AzureServiceBusSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureServiceBusSourceSpec   `json:"spec,omitempty"`
	Status AzureServiceBusSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AzureServiceBusSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*AzureServiceBusSource)(nil)
	_ v1alpha1.EventSource         = (*AzureServiceBusSource)(nil)
	_ v1alpha1.EventSender         = (*AzureServiceBusSource)(nil)
)

// AzureServiceBusSourceSpec defines the desired state of the event source.
type AzureServiceBusSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// The resource ID the Service Bus Topic to subscribe to.
	//
	// Expected format:
	// - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceBus/namespaces/{namespaceName}/topics/{topicName}
	// +optional
	TopicID *AzureResourceID `json:"topicID,omitempty"`

	// The resource ID the Service Bus Queue to subscribe to.
	//
	// Expected format:
	// - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceBus/namespaces/{namespaceName}/queues/{queueName}
	// +optional
	QueueID *AzureResourceID `json:"queueID,omitempty"`

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

// AzureServiceBusSourceStatus defines the observed state of the event source.
type AzureServiceBusSourceStatus struct {
	v1alpha1.Status `json:",inline"`

	// Resource ID of the Service Bus Subscription that is currently used
	// by the event source for consuming events from the configured Service
	// Bus.
	SubscriptionID *AzureResourceID `json:"subscriptionID,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureServiceBusSourceList contains a list of event sources.
type AzureServiceBusSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureServiceBusSource `json:"items"`
}
