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
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TektonTarget defines the schema for the Tekton target
type TektonTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the TektonTarget (from the client).
	Spec TektonTargetSpec `json:"spec"`

	// Status communicates the observed state of the TektonTarget (from the controller).
	// +optional
	Status TektonTargetStatus `json:"status,omitempty"`
}

type TektonTargetSpec struct {
	// ReapPolicy dictates the reaping policy to be applied for the target
	// +optional
	ReapPolicy *TektonTargetReapPolicy `json:"reapPolicy,omitempty"`
}

type TektonTargetReapPolicy struct {
	// ReapSuccessAge How long to wait before reaping runs that were successful
	ReapSuccessAge *string `json:"success,omitempty"`
	// ReapFailAge How long to wait before reaping runs that failed
	ReapFailAge *string `json:"fail,omitempty"`
}

// Check the interfaces TektonTarget should be implementing.
var (
	_ runtime.Object            = (*TektonTarget)(nil)
	_ kmeta.OwnerRefable        = (*TektonTarget)(nil)
	_ targets.IntegrationTarget = (*TektonTarget)(nil)
	_ targets.EventSource       = (*TektonTarget)(nil)
	_ duckv1.KRShaped           = (*TektonTarget)(nil)
)

// TektonTargetStatus communicates the observed state of the TektonTarget (from the controller).
type TektonTargetStatus struct {
	// inherits duck/v1beta1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last
	//   processed by the controller.
	// * Conditions - the latest available observations of a resource's current
	//   state.
	duckv1.Status `json:",inline"`

	// AddressStatus fulfills the Addressable contract.
	duckv1.AddressStatus `json:",inline"`

	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TektonBuildRequestTargetList is a list of TektonBuildRequestTarget resources
type TektonTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TektonTarget `json:"items"`
}
