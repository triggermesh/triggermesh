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
	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// {{.UppercaseName}} is the Schema the event target.
type {{.UppercaseName}} struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   {{.UppercaseName}}Spec   `json:"spec"`
	Status {{.UppercaseName}}Status `json:"status,omitempty"`
}

// Check the interfaces {{.UppercaseName}} should be implementing.
var (
	_ runtime.Object            = (*{{.UppercaseName}})(nil)
	_ kmeta.OwnerRefable        = (*{{.UppercaseName}})(nil)
	_ targets.IntegrationTarget = (*{{.UppercaseName}})(nil)
	_ targets.EventSource       = (*{{.UppercaseName}})(nil)
	_ duckv1.KRShaped           = (*{{.UppercaseName}})(nil)
)

// {{.UppercaseName}}Spec holds the desired state of the event target.
type {{.UppercaseName}}Spec struct {

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// {{.UppercaseName}}Status communicates the observed state of the event target. (from the controller).
type {{.UppercaseName}}Status struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`

	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// {{.UppercaseName}}List is a list of event target instances.
type {{.UppercaseName}}List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []{{.UppercaseName}} `json:"items"`
}
