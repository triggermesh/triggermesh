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
	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureSentinelTarget is the Schema the event target.
type AzureSentinelTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureSentinelTargetSpec   `json:"spec"`
	Status AzureSentinelTargetStatus `json:"status,omitempty"`
}

// Check the interfaces AzureSentinelTarget should be implementing.
var (
	_ runtime.Object            = (*AzureSentinelTarget)(nil)
	_ kmeta.OwnerRefable        = (*AzureSentinelTarget)(nil)
	_ targets.IntegrationTarget = (*AzureSentinelTarget)(nil)
	_ targets.EventSource       = (*AzureSentinelTarget)(nil)
	_ duckv1.KRShaped           = (*AzureSentinelTarget)(nil)
)

// AzureSentinelTargetSpec holds the desired state of the event target.
type AzureSentinelTargetSpec struct {

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// AzureSentinelTargetStatus communicates the observed state of the event target. (from the controller).
type AzureSentinelTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`

	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureSentinelTargetList is a list of event target instances.
type AzureSentinelTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AzureSentinelTarget `json:"items"`
}
