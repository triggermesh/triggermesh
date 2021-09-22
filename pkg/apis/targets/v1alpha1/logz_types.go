/*
Copyright (c) 2021 TriggerMesh Inc.

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

// LogzTarget is the Schema for the Logz Target.
type LogzTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LogzTargetSpec   `json:"spec"`
	Status LogzTargetStatus `json:"status,omitempty"`
}

// Check the interfaces LogzTarget should be implementing.
var (
	_ runtime.Object      = (*LogzTarget)(nil)
	_ kmeta.OwnerRefable  = (*LogzTarget)(nil)
	_ targets.EventSource = (*LogzTarget)(nil)
	_ duckv1.KRShaped     = (*LogzTarget)(nil)
)

// LogzTargetSpec holds the desired state of the LogzTarget.
type LogzTargetSpec struct {

	// ShippingToken defines the API token.
	ShippingToken SecretValueFromSource `json:"shippingToken"`

	// LogsListenerURL Defines the Log listener URL
	LogsListenerURL string `json:"logsListenerURL"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// LogzTargetStatus communicates the observed state of the LogzTarget (from the controller).
type LogzTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`
	CloudEventStatus     `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LogzTargetList is a list of LogzTarget resources
type LogzTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []LogzTarget `json:"items"`
}
