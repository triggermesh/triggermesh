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
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type JQTransformation struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the JQTransformation (from the client).
	// +optional
	Spec JQTransformationSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the JQTransformation (from the controller).
	// +optional
	Status JQTransformationStatus `json:"status,omitempty"`
}

var (
	_ runtime.Object     = (*JQTransformation)(nil)
	_ kmeta.OwnerRefable = (*JQTransformation)(nil)
	_ duckv1.KRShaped    = (*JQTransformation)(nil)
	_ apis.Validatable   = (*XSLTTransformation)(nil)
	_ apis.Defaultable   = (*XSLTTransformation)(nil)
)

// JQTransformationSpec holds the desired state of the JQTransformation (from the client).
type JQTransformationSpec struct {
	// The query that gets passed to the JQ library
	Query string `json:"query"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`

	// Sink is a reference to an object that will resolve to a uri to use as the sink.
	// +optional
	Sink *duckv1.Destination `json:"sink,omitempty"`
}

// JQTransformationStatus communicates the observed state of the JQTransformation (from the controller).
type JQTransformationStatus struct {
	duckv1.SourceStatus  `json:",inline"`
	duckv1.AddressStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JQTransformationList is a list of JQTransformation resources
type JQTransformationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []JQTransformation `json:"items"`
}
