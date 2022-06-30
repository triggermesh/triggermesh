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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaTarget is the Schema for an KafkaTarget.
type KafkaTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaTargetSpec `json:"spec"`
	Status v1alpha1.Status `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*KafkaTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*KafkaTarget)(nil)
)

// KafkaTargetSpec defines the desired state of the event target.
type KafkaTargetSpec struct {
	// Topic where messages are produced.
	Topic string `json:"topic"`

	// TopicReplicationFactor is the number of replicas for the topic.
	// +optional
	TopicReplicationFactor *int `json:"topicReplicationFactor,omitempty"`

	// TopicPartitions is the number of partitions for the topic.
	// +optional
	TopicPartitions *int `json:"topicPartitions,omitempty"`

	// BootstrapServers holds the name of the Kafka Bootstrap server.
	BootstrapServers []string `json:"bootstrapServers"`

	// Auth contains Authentication method used to interact with Kafka.
	Auth KafkaTargetAuth `json:"auth"`

	// Whether to omit CloudEvent context attributes in messages sent to Kafka.
	// When this property is false (default), the entire CloudEvent payload is included.
	// When this property is true, only the CloudEvent data is included.
	DiscardCEContext bool `json:"discardCloudEventContext"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// KafkaTargetAuth contains Authentication method used to interact with Kafka.
type KafkaTargetAuth struct {
	KerberosAuth *KafkaTargetKerberosAuth `json:"kerberosAuth,omitempty"`
	SSLAuth      *KafkaTargetSSLAuth      `json:"sslAuth,omitempty"`

	// SASL Enable
	SASLEnable bool `json:"saslEnable"`

	// TLS Enable
	TLSEnable bool `json:"tlsEnable"`

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

// KafkaTargetSSLAuth contains kerberos credentials.
type KafkaTargetSSLAuth struct {
	SSLCA              *v1alpha1.ValueFromField `json:"sslCA,omitempty"`
	SSLClientCert      *v1alpha1.ValueFromField `json:"sslClientCert,omitempty"`
	SSLClientKey       *v1alpha1.ValueFromField `json:"sslClientKey,omitempty"`
	InsecureSkipVerify *bool                    `json:"insecureSkipVerify,omitempty"`
}

// KafkaTargetKerberosAuth contains kerberos credentials.
type KafkaTargetKerberosAuth struct {
	Username            *string                  `json:"username,omitempty"`
	Password            *v1alpha1.ValueFromField `json:"password,omitempty"`
	KerberosServiceName *string                  `json:"kerberosServiceName,omitempty"`
	KerberosConfigPath  *string                  `json:"kerberosConfigPath,omitempty"`
	KerberosKeytabPath  *string                  `json:"kerberosKeytabPath,omitempty"`
	KerberosConfig      *v1alpha1.ValueFromField `json:"kerberosConfig,omitempty"`
	KerberosKeytab      *v1alpha1.ValueFromField `json:"kerberosKeytab,omitempty"`
	KerberosRealm       *string                  `json:"kerberosRealm,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaTargetList is a list of event target instances.
type KafkaTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KafkaTarget `json:"items"`
}
