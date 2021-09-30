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

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HasuraTarget is the Schema for the event target.
type HasuraTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HasuraTargetSpec   `json:"spec,omitempty"`
	Status HasuraTargetStatus `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ runtime.Object            = (*HasuraTarget)(nil)
	_ kmeta.OwnerRefable        = (*HasuraTarget)(nil)
	_ targets.IntegrationTarget = (*HasuraTarget)(nil)
	_ targets.EventSource       = (*HasuraTarget)(nil)
	_ duckv1.KRShaped           = (*HasuraTarget)(nil)
)

// HasuraTargetSpec defines the desired state of the event target.
type HasuraTargetSpec struct {
	// The GraphQL server endpoint.
	Endpoint string `json:"endpoint"`
	// A user token for interfacing with Hasura.
	// +optional
	JwtToken *SecretValueFromSource `json:"jwt,omitempty"`
	// An alternate token for interfacing with Hasura using admin privileges.
	// +optional
	AdminToken *SecretValueFromSource `json:"admin,omitempty"`
	// A default role that the queries should use when running the query.
	// +optional
	DefaultRole *string `json:"defaultRole,omitempty"`
	// A predefined list of queries that an event can specify in the io.triggermesh.graphql.query event type.
	// +optional
	Queries *map[string]string `json:"queries,omitempty"`
}

// HasuraTargetStatus defines the observed state of the event target.
type HasuraTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`

	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HasuraTargetList contains a list of event targets.
type HasuraTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HasuraTarget `json:"items"`
}
