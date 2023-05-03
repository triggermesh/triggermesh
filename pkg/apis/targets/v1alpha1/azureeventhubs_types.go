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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureEventHubsTarget is the Schema for an Azure Object Storage Service Target.
type AzureEventHubsTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureEventHubsTargetSpec `json:"spec"`
	Status v1alpha1.Status          `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AzureEventHubsTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*AzureEventHubsTarget)(nil)
	_ v1alpha1.EventReceiver       = (*AzureEventHubsTarget)(nil)
	_ v1alpha1.EventSource         = (*AzureEventHubsTarget)(nil)
)

// AzureEventHubsTargetSpec defines the desired state of the event target.
type AzureEventHubsTargetSpec struct {
	// Authentication method to interact with the Azure Event Hubs API.
	Auth AzureAuth `json:"auth"`

	// Resource ID of the Event Hubs instance.
	//
	// Expected format:
	// - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}
	EventHubID AzureResourceID `json:"eventHubID"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`

	DiscardCEContext bool `json:"discardCloudEventContext"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureEventHubsTargetList is a list of event target instances.
type AzureEventHubsTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AzureEventHubsTarget `json:"items"`
}
