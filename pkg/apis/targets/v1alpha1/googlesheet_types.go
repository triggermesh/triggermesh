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

// GoogleSheetTarget is the Schema for an GoogleSheet Target.
type GoogleSheetTarget struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the GoogleSheetTarget (from the client).
	Spec GoogleSheetTargetSpec `json:"spec"`

	// Status communicates the observed state of the GoogleSheetTarget (from the controller).
	// +optional
	Status GoogleSheetTargetStatus `json:"status,omitempty"`
}

// Check the interfaces GoogleSheetTarget should be implementing.
var (
	_ runtime.Object            = (*GoogleSheetTarget)(nil)
	_ kmeta.OwnerRefable        = (*GoogleSheetTarget)(nil)
	_ targets.IntegrationTarget = (*GoogleSheetTarget)(nil)
	_ targets.EventSource       = (*GoogleSheetTarget)(nil)
	_ duckv1.KRShaped           = (*GoogleSheetTarget)(nil)
)

const (
	// GoogleSheetTargetEventType is the GoogleSheetTarget CloudEvent type.
	GoogleSheetTargetEventType = "dev.knative.source.GoogleSheet"
)

// GoogleSheetTargetSpec holds the desired state of the GoogleSheetTarget.
type GoogleSheetTargetSpec struct {
	// GoogleSheet credential JSON for auth
	GoogleServiceAccount SecretValueFromSource `json:"googleServiceAccount"`

	// ID of Google a spreadsheet
	ID string `json:"id"`

	// DefaultPrefix is a pre-defined prefix for the individual sheets.
	DefaultPrefix string `json:"defaultPrefix"`
}

// GoogleSheetTargetStatus communicates the observed state of the GoogleSheetTarget (from the controller).
type GoogleSheetTargetStatus struct {
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

// GoogleSheetTargetList is a list of GoogleSheetTarget resources
type GoogleSheetTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []GoogleSheetTarget `json:"items"`
}
