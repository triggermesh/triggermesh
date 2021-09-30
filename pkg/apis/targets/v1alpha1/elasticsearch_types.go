/*
Copyright (c) 2021 TriggerMesh Inc.

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
	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticsearchTarget is the Schema for an Elasticsearch Target.
type ElasticsearchTarget struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the ElasticsearchTarget (from the client).
	Spec ElasticsearchTargetSpec `json:"spec"`

	// Status communicates the observed state of the ElasticsearchTarget (from the controller).
	// +optional
	Status ElasticsearchTargetStatus `json:"status,omitempty"`
}

// Check the interfaces ElasticsearchTarget should be implementing.
var (
	_ runtime.Object            = (*ElasticsearchTarget)(nil)
	_ kmeta.OwnerRefable        = (*ElasticsearchTarget)(nil)
	_ targets.IntegrationTarget = (*ElasticsearchTarget)(nil)
	_ targets.EventSource       = (*ElasticsearchTarget)(nil)
	_ duckv1.KRShaped           = (*ElasticsearchTarget)(nil)
)

// ElasticsearchTargetSpec holds the desired state of the ElasticsearchTarget.
type ElasticsearchTargetSpec struct {
	// Connection information to elasticsearch.
	// +optional
	Connection Connection `json:"connection"`

	// IndexName to write to.
	IndexName string `json:"indexName"`

	// Whether to omit CloudEvent context attributes in created documents.
	// When this property is false (default), the entire CloudEvent payload is included.
	// When this property is true, only the CloudEvent data is included.
	DiscardCEContext bool `json:"discardCloudEventContext"`

	// EventOptions for targets.
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// Connection contains connection and configuration parameters
type Connection struct {
	// Array of hostnames or IP addresses to connect the target to.
	Addresses []string `json:"addresses,omitempty"`
	// CA Certificate used to verify connection with the Elasticsearch instance.
	CACert *string `json:"caCert,omitempty"`
	// Skip verification of the SSL certificate during the connection.
	SkipVerify *bool `json:"skipVerify,omitempty"`

	// Elasticsearch account username.
	Username *string `json:"username,omitempty"`
	// Elasticsearch account password.
	Password *SecretValueFromSource `json:"password,omitempty"`

	// When informed supersedes username and password.
	APIKey *SecretValueFromSource `json:"apiKey,omitempty"`
}

// ElasticsearchTargetStatus communicates the observed state of the ElasticsearchTarget (from the controller).
type ElasticsearchTargetStatus struct {
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

// ElasticsearchTargetList is a list of ElasticsearchTarget resources
type ElasticsearchTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ElasticsearchTarget `json:"items"`
}
