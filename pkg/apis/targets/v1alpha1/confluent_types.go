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
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfluentTarget is the Schema for an ConfluentTarget.
type ConfluentTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the ConfluentTarget (from the client).
	Spec ConfluentTargetSpec `json:"spec"`

	// Status communicates the observed state of the ConfluentTarget (from the controller).
	Status ConfluentTargetStatus `json:"status,omitempty"`
}

// Check the interfaces ConfluentTarget should be implementing.
var (
	_ runtime.Object     = (*ConfluentTarget)(nil)
	_ kmeta.OwnerRefable = (*ConfluentTarget)(nil)
	_ duckv1.KRShaped    = (*ConfluentTarget)(nil)
)

// ConfluentTargetSpec holds the desired state of the ConfluentTarget.
type ConfluentTargetSpec struct {
	// SASLUsername Confluent account User
	SASLUsername string `json:"username"`

	// SASLPassword Confluent account Password
	SASLPassword SecretValueFromSource `json:"password"`

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
	SASLMechanisms string `json:"saslMechanism"`

	// Whether to omit CloudEvent context attributes in created messages.
	// When this property is false (default), the entire CloudEvent payload is included.
	// When this property is true, only the CloudEvent data is included.
	DiscardCEContext bool `json:"discardCloudEventContext"`
}

// ConfluentTargetStatus communicates the observed state of the ConfluentTarget (from the controller).
type ConfluentTargetStatus struct {
	// inherits duck/v1beta1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last
	//   processed by the controller.
	// * Conditions - the latest available observations of a resource's current
	//   state.
	duckv1.Status `json:",inline"`

	// AddressStatus fulfills the Addressable contract.
	duckv1.AddressStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfluentTargetList is a list of ConfluentTarget resources
type ConfluentTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ConfluentTarget `json:"items"`
}
