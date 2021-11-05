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
	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureEventHubsTarget is the Schema for an Alibaba Object Storage Service Target.
type AzureEventHubsTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureEventHubsTargetSpec   `json:"spec"`
	Status AzureEventHubsTargetStatus `json:"status,omitempty"`
}

// Check the interfaces AzureEventHubsTarget should be implementing.
var (
	_ runtime.Object            = (*AzureEventHubsTarget)(nil)
	_ kmeta.OwnerRefable        = (*AzureEventHubsTarget)(nil)
	_ targets.IntegrationTarget = (*AzureEventHubsTarget)(nil)
	_ targets.EventSource       = (*AzureEventHubsTarget)(nil)
	_ duckv1.KRShaped           = (*AzureEventHubsTarget)(nil)
)

// AzureEventHubsTargetSpec holds the desired state of the AzureEventHubsTarget.
type AzureEventHubsTargetSpec struct {
	// Authentication method to interact with the Azure Event Hubs API.
	Auth AzureAuth `json:"auth"`

	// Resource ID of the Event Hubs instance.
	//
	// Expected format:
	// - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}
	EventHubID EventHubResourceID `json:"eventHubID"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// AzureEventHubsTargetStatus communicates the observed state of the AzureEventHubsTarget (from the controller).
type AzureEventHubsTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`

	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureEventHubsTargetList is a list of AzureEventHubsTarget resources
type AzureEventHubsTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AzureEventHubsTarget `json:"items"`
}
