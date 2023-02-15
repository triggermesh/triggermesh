/*
Copyright 2023 TriggerMesh Inc.

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

// SolaceSource is the Schema for the event source.
type SolaceSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SolaceSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status  `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*SolaceSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*SolaceSource)(nil)
	_ v1alpha1.EventSource         = (*SolaceSource)(nil)
	_ v1alpha1.EventSender         = (*SolaceSource)(nil)
)

// SolaceSourceSpec defines the desired state of the event source.
type SolaceSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// URL
	URL string `json:"url"`

	// QueueName
	QueueName string `json:"queueName"`

	// Auth contains Authentication method used to interact with Solace.
	// +optional
	Auth *SolaceSourceAuth `json:"auth,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// SolaceSourceAuth contains Authentication method used to interact with Solace.
type SolaceSourceAuth struct {
	// TLS
	// +optional
	TLS *SolaceSourceTLSAuth `json:"tls,omitempty"`

	// SASL Enable
	// +optional
	SASLEnable *bool `json:"saslEnable"`

	// TLS Enable
	// +optional
	TLSEnable *bool `json:"tlsEnable,omitempty"`

	// Username Solace
	// +optional
	Username *string `json:"username,omitempty"`

	// Password Solace
	// +optional
	Password *v1alpha1.ValueFromField `json:"password,omitempty"`
}

// SolaceSourceTLSAuth contains kerberos credentials.
type SolaceSourceTLSAuth struct {
	CA         *v1alpha1.ValueFromField `json:"ca,omitempty"`
	ClientCert *v1alpha1.ValueFromField `json:"clientCert,omitempty"`
	ClientKey  *v1alpha1.ValueFromField `json:"clientKey,omitempty"`
	SkipVerify *bool                    `json:"skipVerify,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SolaceSourceList contains a list of event sources.
type SolaceSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SolaceSource `json:"items"`
}
