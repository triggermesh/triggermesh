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

	// Topic where messages are produced.
	Topic string `json:"topic"`

	// TopicReplicationFactor is the number of replicas for the topic.
	// +optional
	TopicReplicationFactor *int `json:"topicReplicationFactor"`

	// TopicPartitions is the number of partitions for the topic.
	// +optional
	TopicPartitions *int `json:"topicPartitions"`

	// BootstrapServers holds the name of the Kafka Bootstrap server.
	BootstrapServers []string `json:"bootstrapServers"`

	// SecurityProtocol allows the user to set the security protocol
	SecurityProtocol string `json:"securityProtocol"`

	// SASLMechanisms all the assignment of specific SASL mechanisms.
	SecurityMechanisms string `json:"securityMechanism"`

	// SSL Authentication method to interact with Kafka.
	SSLAuth KafkaTargetSSLAuth `json:"sslAuth"`

	// Kerberos Authentication method to interact with Kafka.
	KerberosAuth KafkaTargetKerberosAuth `json:"kerberosAuth"`

	// Whether to omit CloudEvent context attributes in messages sent to Kafka.
	// When this property is false (default), the entire CloudEvent payload is included.
	// When this property is true, only the CloudEvent data is included.
	DiscardCEContext bool `json:"discardCloudEventContext"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// KafkaTargetKerberosAuth contains kerberos credentials.
type KafkaTargetSSLAuth struct {
	SSLCA         *v1alpha1.ValueFromField `json:"sslCA"`
	SSLClientCert *v1alpha1.ValueFromField `json:"sslClientCert"`
	SSLClientKey  *v1alpha1.ValueFromField `json:"sslClientKey"`
}

// KafkaTargetKerberosAuth contains kerberos credentials.
type KafkaTargetKerberosAuth struct {
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

// KafkaTargetList is a list of event target instances.
type KafkaTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KafkaTarget `json:"items"`
}
