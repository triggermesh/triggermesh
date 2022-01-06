/*
Copyright 2020 TriggerMesh Inc.

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

// XMLToJSONTransformation is a Knative abstraction that encapsulates the interface by which Knative
// components express a desire to have a particular image cached.
type XMLToJSONTransformation struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the XMLToJSONTransformation (from the client).
	// +optional
	Spec XMLToJSONTransformationSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the XMLToJSONTransformation (from the controller).
	// +optional
	Status XMLToJSONTransformationStatus `json:"status,omitempty"`
}

var (
	// Check that XMLToJSONTransformation can be validated and defaulted.
	_ apis.Validatable   = (*XMLToJSONTransformation)(nil)
	_ apis.Defaultable   = (*XMLToJSONTransformation)(nil)
	_ kmeta.OwnerRefable = (*XMLToJSONTransformation)(nil)
	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*XMLToJSONTransformation)(nil)
)

// XMLToJSONTransformationSpec holds the desired state of the XMLToJSONTransformation (from the client).
type XMLToJSONTransformationSpec struct {
	// Sink is a reference to an object that will resolve to a uri to use as the sink.
	Sink duckv1.Destination `json:"sink,omitempty"`
}

const (
	// XMLToJSONTransformationConditionReady is set when the revision is starting to materialize
	// runtime resources, and becomes true when those resources are ready.
	XMLToJSONTransformationConditionReady = apis.ConditionReady
)

// XMLToJSONTransformationStatus communicates the observed state of the XMLToJSONTransformation (from the controller).
type XMLToJSONTransformationStatus struct {
	duckv1.SourceStatus `json:",inline"`

	// Address holds the information needed to connect this Addressable up to receive events.
	// +optional
	Address *duckv1.Addressable `json:"address,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// XMLToJSONTransformationList is a list of XMLToJSONTransformation resources
type XMLToJSONTransformationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []XMLToJSONTransformation `json:"items"`
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (t *XMLToJSONTransformation) GetStatus() *duckv1.Status {
	return &t.Status.Status
}
