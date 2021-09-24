/*
Copyright 2021 Triggermesh Inc.

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

// Filter is an addressable object that filters incoming events according
// to provided Common Language Expression
type Filter struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Filter (from the client).
	// +optional
	Spec FilterSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the Filter (from the controller).
	// +optional
	Status RouterStatus `json:"status,omitempty"`
}

var (
	// Check that Filter can be validated and defaulted.
	_ apis.Validatable   = (*Filter)(nil)
	_ apis.Defaultable   = (*Filter)(nil)
	_ kmeta.OwnerRefable = (*Filter)(nil)
	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*Filter)(nil)
	_ multiTenant     = (*Filter)(nil)

	_ Router = (*Filter)(nil)
)

// FilterSpec contains CEL expression string and the destination sink
type FilterSpec struct {
	Expression string `json:"expression"`

	// Sink is a reference to an object that will resolve to a domain name to use as the sink.
	Sink *duckv1.Destination `json:"sink"`
}

// FilterList is a list of Filter resources
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FilterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Filter `json:"items"`
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (f *Filter) GetStatus() *duckv1.Status {
	return &f.Status.Status
}

// AsRouter implements Router.
func (f *Filter) AsRouter() string {
	return "filter/" + f.Name
}
