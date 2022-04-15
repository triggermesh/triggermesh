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

// AzureIOTHubSource is the Schema for the event source.
type AzureIOTHubSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureIOTHubSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status       `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AzureIOTHubSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*AzureIOTHubSource)(nil)
	_ v1alpha1.EventSource         = (*AzureIOTHubSource)(nil)
	_ v1alpha1.EventSender         = (*AzureIOTHubSource)(nil)
)

// AzureIOTHubSourceSpec defines the desired state of the event source.
type AzureIOTHubSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// AzureAuth contains multiple authentication methods for Azure services.
	Auth AzureAuth `json:"auth,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureIOTHubSourceList contains a list of event sources.
type AzureIOTHubSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureIOTHubSource `json:"items"`
}
