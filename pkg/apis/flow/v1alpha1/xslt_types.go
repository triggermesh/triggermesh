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
	"k8s.io/apimachinery/pkg/runtime"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// XsltTransform is the Schema for an XSLT transformation target.
type XsltTransform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the XsltTransform object.
	Spec XsltTransformSpec `json:"spec"`

	// Status communicates the observed state of the XsltTransform object.
	// +optional
	Status XsltTransformStatus `json:"status,omitempty"`
}

// Check the interfaces XsltTransform should be implementing.
var (
	_ runtime.Object     = (*XsltTransform)(nil)
	_ kmeta.OwnerRefable = (*XsltTransform)(nil)
	_ duckv1.KRShaped    = (*XsltTransform)(nil)
	_ apis.Validatable   = (*XsltTransform)(nil)
	_ apis.Defaultable   = (*XsltTransform)(nil)
)

// XsltTransformSpec holds the desired state of the XsltTransform.
type XsltTransformSpec struct {
	// XSLT document that will be used by default for transformation.
	// Can be omited if the XSLT is informed at each event.
	// +optional
	XSLT *ValueFromField `json:"xslt,omitempty"`

	// Whether the default XSLT can be overriden at each event
	// +optional
	AllowPerEventXSLT *bool `json:"allowPerEventXslt,omitempty"`
}

// XsltTransformStatus communicates the observed state of the component.
type XsltTransformStatus struct {
	// Although this is not a source, it is a CloudEvents producing entity.
	duckv1.SourceStatus `json:",inline"`
	// AddressStatus fulfills the Addressable contract.
	// +optional
	duckv1.AddressStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// XsltTransformList is a list of XsltTransform resources
type XsltTransformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []XsltTransform `json:"items"`
}
