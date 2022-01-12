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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureServiceBusQueueSource is the Schema for the event source.
type AzureServiceBusQueueSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureServiceBusQueueSourceSpec `json:"spec,omitempty"`
	Status EventSourceStatus              `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ EventSource = (*AzureServiceBusQueueSource)(nil)
)

// AzureServiceBusQueueSourceSpec defines the desired state of the event source.
type AzureServiceBusQueueSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// The resource ID the Service Bus Queue to subscribe to.
	//
	// Expected format:
	// - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceBus/namespaces/{namespaceName}/queues/{queueName}
	QueueID AzureResourceID `json:"queueID"`

	// Authentication method to interact with Azure Service Bus.
	Auth AzureAuth `json:"auth"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureServiceBusQueueSourceList contains a list of event sources.
type AzureServiceBusQueueSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureServiceBusQueueSource `json:"items"`
}
