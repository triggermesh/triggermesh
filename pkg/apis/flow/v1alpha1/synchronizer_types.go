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

	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis"
	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Synchronizer is the Schema for the Synchronizer target.
type Synchronizer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SynchronizerSpec `json:"spec"`
	Status v1alpha1.Status  `json:"status,omitempty"`
}

// Check the interfaces Synchronizer should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*Synchronizer)(nil)
	_ v1alpha1.AdapterConfigurable = (*Synchronizer)(nil)
	_ v1alpha1.EventSender         = (*Synchronizer)(nil)
)

// SynchronizerSpec defines the desired state of the component.
type SynchronizerSpec struct {
	CorrelationKey Correlation `json:"correlationKey"`
	Response       Response    `json:"response"`

	// Support sending to an event sink instead of replying.
	duckv1.SourceSpec `json:",inline"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// Correlation holds the request-response matching parameters.
type Correlation struct {
	Attribute string `json:"attribute"`
	Length    int    `json:"length"`
}

// Response defines the response handling configuration.
type Response struct {
	Timeout apis.Duration `json:"timeout"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SynchronizerList is a list of component instances.
type SynchronizerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Synchronizer `json:"items"`
}
