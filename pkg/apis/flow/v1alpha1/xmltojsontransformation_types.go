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
	"github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// XMLToJSONTransformation is the schema for the event transformer.
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
	_ kmeta.OwnerRefable = (*XMLToJSONTransformation)(nil)
	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*XMLToJSONTransformation)(nil)
)

// XMLToJSONTransformationSpec holds the desired state of the XMLToJSONTransformation (from the client).
type XMLToJSONTransformationSpec struct {
	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`

	// Sink is a reference to an object that will resolve to a uri to use as the sink.
	Sink duckv1.Destination `json:"sink,omitempty"`
}

// EventOptions modifies CloudEvents management at Targets.
type EventOptions struct {
	// PayloadPolicy indicates if replies from the target should include
	// a payload if available. Possible values are:
	//
	// - always: will return a with the reply payload if avaliable.
	// - errors: will only reply with payload in case of an error.
	// - never: will not reply with payload.
	//
	// +optional
	PayloadPolicy *cloudevents.PayloadPolicy `json:"payloadPolicy,omitempty"`
}

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
