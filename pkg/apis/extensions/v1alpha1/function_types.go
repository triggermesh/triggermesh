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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Function is an addressable object that executes function code.
type Function struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FunctionSpec   `json:"spec,omitempty"`
	Status FunctionStatus `json:"status,omitempty"`
}

var (
	_ v1alpha1.Reconcilable        = (*Function)(nil)
	_ v1alpha1.AdapterConfigurable = (*Function)(nil)
	_ v1alpha1.EventSource         = (*Function)(nil)
	_ v1alpha1.EventSender         = (*Function)(nil)
)

// FunctionSpec holds the desired state of the Function Specification
type FunctionSpec struct {
	Runtime         string               `json:"runtime"`
	Entrypoint      string               `json:"entrypoint"`
	Code            string               `json:"code"`
	ResponseIsEvent bool                 `json:"responseIsEvent,omitempty"`
	EventStore      EventStoreConnection `json:"eventStore,omitempty"`

	// Support sending to an event sink instead of replying,
	// as well as setting the CloudEvents 'type' and 'source' attributes
	// using CloudEventOverrides (hack).
	duckv1.SourceSpec `json:",inline"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// EventStoreConnection contains the data to connect to
// an EventStore instance
type EventStoreConnection struct {
	// URI is the gRPC location to the EventStore
	URI string `json:"uri"`
}

// FunctionStatus defines the observed state of the Function.
type FunctionStatus struct {
	v1alpha1.Status `json:",inline"`
	ConfigMap       *FunctionConfigMapIdentity `json:"configMap,omitempty"`
}

// FunctionConfigMapIdentity represents the identity of the ConfigMap
// containing the code of a Function.
type FunctionConfigMapIdentity struct {
	Name            string `json:"name"`
	ResourceVersion string `json:"resourceVersion"`
}

// FunctionList is a list of Function resources
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Function `json:"items"`
}
