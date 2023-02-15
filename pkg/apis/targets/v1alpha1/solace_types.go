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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SolaceTarget is the Schema for an SolaceTarget.
type SolaceTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SolaceTargetSpec `json:"spec"`
	Status v1alpha1.Status  `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*SolaceTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*SolaceTarget)(nil)
)

// SolaceTargetSpec defines the desired state of the event target.
type SolaceTargetSpec struct {
	// URL
	URL string `json:"url"`

	// QueueName
	QueueName string `json:"queueName"`

	// Auth contains Authentication method used to interact with Solace.
	// +optional
	Auth *SolaceTargetAuth `json:"auth,omitempty"`

	// Whether to omit CloudEvent context attributes in messages sent to Solace.
	// When this property is false (default), the entire CloudEvent payload is included.
	// When this property is true, only the CloudEvent data is included.
	DiscardCEContext bool `json:"discardCloudEventContext"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// SolaceTargetAuth contains Authentication method used to interact with Solace.
type SolaceTargetAuth struct {
	// TLS
	// +optional
	TLS *SolaceTargetTLSAuth `json:"tls,omitempty"`

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

// SolaceTargetTLSAuth contains SSL credentials.
type SolaceTargetTLSAuth struct {
	CA         *v1alpha1.ValueFromField `json:"ca,omitempty"`
	ClientCert *v1alpha1.ValueFromField `json:"clientCert,omitempty"`
	ClientKey  *v1alpha1.ValueFromField `json:"clientKey,omitempty"`
	SkipVerify *bool                    `json:"skipVerify,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SolaceTargetList is a list of event target instances.
type SolaceTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []SolaceTarget `json:"items"`
}
