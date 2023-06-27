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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureServiceBusTarget is the Schema for an Azure Service Bus Target.
type AzureServiceBusTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureServiceBusTargetSpec `json:"spec"`
	Status v1alpha1.Status           `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AzureServiceBusTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*AzureServiceBusTarget)(nil)
	_ v1alpha1.EventReceiver       = (*AzureServiceBusTarget)(nil)
	_ v1alpha1.EventSource         = (*AzureServiceBusTarget)(nil)
)

// AzureServiceBusTargetSpec defines the desired state of the event target.
type AzureServiceBusTargetSpec struct {
	// The resource ID the Service Bus Topic.
	//
	// Expected format:
	// - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceBus/namespaces/{namespaceName}/topics/{topicName}
	// +optional
	TopicID *AzureResourceID `json:"topicID,omitempty"`

	// The resource ID the Service Bus Queue.
	//
	// Expected format:
	// - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceBus/namespaces/{namespaceName}/queues/{queueName}
	// +optional
	QueueID *AzureResourceID `json:"queueID,omitempty"`

	// Authentication method to interact with the Azure Service Bus API.
	Auth AzureAuth `json:"auth"`

	// WebSocketsEnable
	// +optional
	WebSocketsEnable *bool `json:"webSocketsEnable,omitempty"`

	// EventOptions for targets
	// +optional
	EventOptions *EventOptions `json:"eventOptions,omitempty"`

	DiscardCEContext bool `json:"discardCloudEventContext"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureServiceBusTargetList is a list of event target instances.
type AzureServiceBusTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AzureServiceBusTarget `json:"items"`
}
