/*
Copyright 2020 Triggermesh Inc..

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
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Transformation is a Knative abstraction that encapsulates the interface by which Knative
// components express a desire to have a particular image cached.
type Transformation struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Transformation (from the client).
	// +optional
	Spec TransformationSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the Transformation (from the controller).
	// +optional
	Status TransformationStatus `json:"status,omitempty"`
}

var (
	// Check that Transformation can be validated and defaulted.
	_ apis.Validatable   = (*Transformation)(nil)
	_ apis.Defaultable   = (*Transformation)(nil)
	_ kmeta.OwnerRefable = (*Transformation)(nil)
	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*Transformation)(nil)
)

// TransformationSpec holds the desired state of the Transformation (from the client).
type TransformationSpec struct {
	// Sink is a reference to an object that will resolve to a uri to use as the sink.
	Sink duckv1.Destination `json:"sink,omitempty"`
	// Context contains Transformations that must be applied on CE Context
	Context []Transform `json:"context,omitempty"`
	// Data contains Transformations that must be applied on CE Data
	Data []Transform `json:"data,omitempty"`
}

// Transform describes transformation schemes for different CE types.
type Transform struct {
	Operation string `json:"operation"`
	Paths     []Path `json:"paths"`
}

// Path is a key-value pair that represents JSON object path
type Path struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

const (
	// TransformationConditionReady is set when the revision is starting to materialize
	// runtime resources, and becomes true when those resources are ready.
	TransformationConditionReady = apis.ConditionReady
)

// TransformationStatus communicates the observed state of the Transformation (from the controller).
type TransformationStatus struct {
	duckv1.SourceStatus `json:",inline"`

	// Address holds the information needed to connect this Addressable up to receive events.
	// +optional
	Address *duckv1.Addressable `json:"address,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TransformationList is a list of Transformation resources
type TransformationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Transformation `json:"items"`
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (t *Transformation) GetStatus() *duckv1.Status {
	return &t.Status.Status
}
