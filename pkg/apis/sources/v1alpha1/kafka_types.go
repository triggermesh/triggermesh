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

// KafkaSource is the Schema for the KafkaSource.
type KafkaSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable = (*KafkaSource)(nil)
	_ v1alpha1.EventSender  = (*KafkaSource)(nil)
)

// KafkaSourceSpec defines the desired state of the event source.
type KafkaSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// SALS Enable
	SALSEnable bool `json:"salsEnable"`

	// TLS Enable
	TLSEnable bool `json:"tlsEnable"`

	// Username Kafka account User
	// +optional
	Username *string `json:"username"`

	// Password Kafka account Password
	// +optional
	Password *v1alpha1.ValueFromField `json:"password"`

	// BootstrapServers holds the name of the Kafka Bootstrap server.
	BootstrapServers []string `json:"bootstrapServers"`

	// Topics holds the name of the Kafka Topics.
	Topics []string `json:"topics"`

	// GroupID holds the name of the Kafka Group ID.
	GroupID string `json:"groupID"`

	// SASLMechanisms all the assignment of specific SASL mechanisms.
	SecurityMechanisms string `json:"securityMechanism"`

	// TODO SSLAuth
	SSLAuth KafkaSourceSSLAuth `json:"sslAuth"`

	// TODO KerberosAuth
	KerberosAuth KafkaSourceKerberosAuth `json:"kerberosAuth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// TODO: duplicated SSLCA and check *
// KafkaSourceSSLAuth contains kerberos credentials.
type KafkaSourceSSLAuth struct {
	SSLCA         *v1alpha1.ValueFromField `json:"sslCA"`
	SSLClientCert *v1alpha1.ValueFromField `json:"sslClientCert"`
	SSLClientKey  *v1alpha1.ValueFromField `json:"sslClientKey"`
}

// Keytab needs to be configmap?
// KafkaSourceKerberosAuth contains kerberos credentials.
type KafkaSourceKerberosAuth struct {
	Username            *string                  `json:"username"`
	Password            *v1alpha1.ValueFromField `json:"password"`
	KerberosServiceName *string                  `json:"kerberosServiceName"`
	KerberosConfigPath  *string                  `json:"kerberosConfigPath"`
	KerberosKeytabPath  *string                  `json:"kerberosKeytabPath"`
	KerberosSSLCA       *v1alpha1.ValueFromField `json:"sslCA"`
	KerberosConfig      *v1alpha1.ValueFromField `json:"kerberosConfig"`
	KerberosKeytab      *v1alpha1.ValueFromField `json:"kerberosKeytab"`
	KerberosRealm       *string                  `json:"kerberosRealm"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaSourceList contains a list of event sources.
type KafkaSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaSource `json:"items"`
}
