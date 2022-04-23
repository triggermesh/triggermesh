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

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// XSLTTransformation is the Schema for an XSLT transformation target.
type XSLTTransformation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   XSLTTransformationSpec `json:"spec"`
	Status v1alpha1.Status        `json:"status,omitempty"`
}

// Check the interfaces XSLTTransformation should be implementing.
var (
	_ apis.Validatable = (*XSLTTransformation)(nil)
	_ apis.Defaultable = (*XSLTTransformation)(nil)

	_ v1alpha1.Reconcilable        = (*XSLTTransformation)(nil)
	_ v1alpha1.AdapterConfigurable = (*XSLTTransformation)(nil)
	_ v1alpha1.EventSender         = (*XSLTTransformation)(nil)
)

// XSLTTransformationSpec defines the desired state of the component.
type XSLTTransformationSpec struct {
	// XSLT document that will be used by default for transformation.
	// Can be omited if the XSLT is informed at each event.
	// +optional
	XSLT *ValueFromField `json:"xslt,omitempty"`

	// Whether the default XSLT can be overriden at each event
	// +optional
	AllowPerEventXSLT *bool `json:"allowPerEventXSLT,omitempty"`

	// Support sending to an event sink instead of replying.
	duckv1.SourceSpec `json:",inline"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// XSLTTransformationList is a list of component instances.
type XSLTTransformationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []XSLTTransformation `json:"items"`
}
