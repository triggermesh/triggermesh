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

// Splitter is an addressable object that splits incoming events according
// to provided specification
type Splitter struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Splitter (from the client).
	// +optional
	Spec SplitterSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the Splitter (from the controller).
	// +optional
	Status RouterStatus `json:"status,omitempty"`
}

var (
	// Check that Splitter can be validated and defaulted.
	_ apis.Validatable   = (*Splitter)(nil)
	_ apis.Defaultable   = (*Splitter)(nil)
	_ kmeta.OwnerRefable = (*Splitter)(nil)
	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*Splitter)(nil)
	_ multiTenant     = (*Splitter)(nil)

	_ Router = (*Splitter)(nil)
)

// SplitterSpec holds the desired state of the Splitter
type SplitterSpec struct {
	Path      string              `json:"path"`
	CEContext CloudEventContext   `json:"ceContext"`
	Sink      *duckv1.Destination `json:"sink"`
}

type CloudEventContext struct {
	Type       string            `json:"type"`
	Source     string            `json:"source"`
	Extensions map[string]string `json:"extensions"`
}

// SplitterList is a list of Splitter resources
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SplitterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Splitter `json:"items"`
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *Splitter) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// AsRouter implements Router.
func (s *Splitter) AsRouter() string {
	return "splitter/" + s.Name
}
