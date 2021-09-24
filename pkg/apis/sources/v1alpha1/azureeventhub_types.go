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
	"k8s.io/apimachinery/pkg/runtime"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureEventHubSource is the Schema for the event source.
type AzureEventHubSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureEventHubSourceSpec `json:"spec,omitempty"`
	Status EventSourceStatus       `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ runtime.Object   = (*AzureEventHubSource)(nil)
	_ apis.Validatable = (*AzureEventHubSource)(nil)
	_ apis.Defaultable = (*AzureEventHubSource)(nil)
	_ EventSource      = (*AzureEventHubSource)(nil)
)

// AzureEventHubSourceSpec defines the desired state of the event source.
type AzureEventHubSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// The attributes below identify the Event Hubs instance.
	// Both are optional, because this information may alternatively come
	// from the SAS token's connection string, when this mode of
	// authentication is selected.
	//
	// Event Hubs namespace containing the Event Hubs instance to source events from.
	// +optional
	HubNamespace string `json:"hubNamespace,omitempty"`
	// Event Hubs instance to source events from.
	// +optional
	HubName string `json:"hubName,omitempty"`

	// Authentication method to interact with the Azure Event Hubs API.
	Auth AzureAuth `json:"auth"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureEventHubSourceList contains a list of event sources.
type AzureEventHubSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureEventHubSource `json:"items"`
}
