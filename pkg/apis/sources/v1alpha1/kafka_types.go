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
	_ v1alpha1.Reconcilable        = (*KafkaSource)(nil)
	_ v1alpha1.EventSender         = (*KafkaSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*KafkaSource)(nil)
	_ v1alpha1.EventSource         = (*KafkaSource)(nil)
)

// KafkaSourceSpec defines the desired state of the event source.
type KafkaSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// BootstrapServers holds the name of the Kafka Bootstrap server.
	BootstrapServers []string `json:"bootstrapServers"`

	// Topic holds the name of the Kafka Topic.
	Topic string `json:"topic"`

	// GroupID holds the name of the Kafka Group ID.
	GroupID string `json:"groupID"`

	// Auth contains Authentication method used to interact with Kafka.
	// +optional
	Auth KafkaSourceAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// KafkaSourceAuth contains Authentication method used to interact with Kafka.
type KafkaSourceAuth struct {
	Kerberos *KafkaSourceKerberos `json:"kerberos,omitempty"`
	TLS      *KafkaSourceTLSAuth  `json:"tls,omitempty"`

	// SASL Enable
	SASLEnable bool `json:"saslEnable"`

	// TLS Enable
	// +optional
	TLSEnable *bool `json:"tlsEnable,omitempty"`

	// SecurityMechanisms holds the assignment of the specific SASL mechanisms.
	// +optional
	SecurityMechanisms *string `json:"securityMechanism,omitempty"`

	// Username Kafka account User
	// +optional
	Username *string `json:"username,omitempty"`

	// Password Kafka account Password
	// +optional
	Password *v1alpha1.ValueFromField `json:"password,omitempty"`
}

// KafkaSourceTLSAuth contains kerberos credentials.
type KafkaSourceTLSAuth struct {
	CA         *v1alpha1.ValueFromField `json:"ca,omitempty"`
	ClientCert *v1alpha1.ValueFromField `json:"clientCert,omitempty"`
	ClientKey  *v1alpha1.ValueFromField `json:"clientKey,omitempty"`
	SkipVerify *bool                    `json:"skipVerify,omitempty"`
}

// KafkaSourceKerberos contains kerberos credentials.
type KafkaSourceKerberos struct {
	Username    *string                  `json:"username,omitempty"`
	Password    *v1alpha1.ValueFromField `json:"password,omitempty"`
	Realm       *string                  `json:"realm,omitempty"`
	ServiceName *string                  `json:"serviceName,omitempty"`
	ConfigPath  *string                  `json:"configPath,omitempty"`
	KeytabPath  *string                  `json:"keytabPath,omitempty"`
	Config      *v1alpha1.ValueFromField `json:"config,omitempty"`
	Keytab      *v1alpha1.ValueFromField `json:"keytab,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaSourceList contains a list of event sources.
type KafkaSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaSource `json:"items"`
}
