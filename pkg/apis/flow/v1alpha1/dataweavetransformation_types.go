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

// DataWeaveTransformation is the Schema for an DataWeave transformation target.
type DataWeaveTransformation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataWeaveTransformationSpec `json:"spec"`
	Status v1alpha1.Status             `json:"status,omitempty"`
}

var (
	_ v1alpha1.Reconcilable        = (*DataWeaveTransformation)(nil)
	_ v1alpha1.AdapterConfigurable = (*DataWeaveTransformation)(nil)
	_ v1alpha1.EventSender         = (*DataWeaveTransformation)(nil)
)

// DataWeaveTransformationSpec defines the desired state of the component.
type DataWeaveTransformationSpec struct {
	// DataWeave spell that will be used by default for transformation.
	// +optional
	DwSpell *ValueFromField `json:"dwSpell,omitempty"`

	// Whether the default DwSpell can be overriden at each event
	// +optional
	AllowPerEventDwSpell *bool `json:"allowPerEventDwSpell,omitempty"`

	// Content type for the incoming transformation.
	// +optional
	InputContentType *string `json:"inputContentType,omitempty"`

	// Content type for transformation Output.
	// +optional
	OutputContentType *string `json:"outputContentType,omitempty"`

	// Support sending to an event sink instead of replying.
	duckv1.SourceSpec `json:",inline"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DataWeaveTransformationList is a list of component instances.
type DataWeaveTransformationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []DataWeaveTransformation `json:"items"`
}
