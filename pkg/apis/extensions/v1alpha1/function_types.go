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
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Function is an addressable object that executes function code.
type Function struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Function (from the client).
	// +optional
	Spec FunctionSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the Function (from the controller).
	// +optional
	Status FunctionStatus `json:"status,omitempty"`
}

var (
	// Check that Function can be validated and defaulted.
	_ apis.Validatable   = (*Function)(nil)
	_ apis.Defaultable   = (*Function)(nil)
	_ kmeta.OwnerRefable = (*Function)(nil)
	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*Function)(nil)
)

// FunctionSpec holds the desired state of the Function Specification
type FunctionSpec struct {
	Runtime             string                      `json:"runtime"`
	Entrypoint          string                      `json:"entrypoint"`
	Public              bool                        `json:"public,omitempty"`
	Code                string                      `json:"code"`
	ResponseIsEvent     bool                        `json:"responseIsEvent,omitempty"`
	EventStore          EventStoreConnection        `json:"eventStore,omitempty"`
	CloudEventOverrides *duckv1.CloudEventOverrides `json:"ceOverrides"`
	Sink                *duckv1.Destination         `json:"sink"`
}

// EventStoreConnection contains the data to connect to
// an EventStore instance
type EventStoreConnection struct {
	// URI is the gRPC location to the EventStore
	URI string `json:"uri"`
}

// FunctionStatus communicates the observed state of the Function (from the controller).
type FunctionStatus struct {
	duckv1.SourceStatus `json:",inline"`

	// Address holds the information needed to connect this Function up to receive events.
	// +optional
	Address *duckv1.Addressable `json:"address,omitempty"`
}

// FunctionList is a list of Function resources
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Function `json:"items"`
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (f *Function) GetStatus() *duckv1.Status {
	return &f.Status.Status
}
