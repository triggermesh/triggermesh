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

// InfraTarget is the Schema for the Infra JS Target.
type InfraTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InfraTargetSpec `json:"spec"`
	Status v1alpha1.Status `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable = (*InfraTarget)(nil)
)

// InfraTargetSpec holds the desired state of the InfraTarget.
type InfraTargetSpec struct {
	// Script to be executed at every request.
	Script *InfraTargetScript `json:"script,omitempty"`

	// State actions and options.
	State *InfraTargetState `json:"state,omitempty"`

	// TypeLoopProtection protect against infinite loops when the cloudevent type does not change.
	TypeLoopProtection *bool `json:"typeLoopProtection,omitempty"`
}

// InfraTargetScript holds the script options
type InfraTargetScript struct {
	// Code to be executed at every request.
	Code string `json:"code"`

	// Timeout is the script execution time after which
	// it will be halted.
	Timeout *int `json:"timeout,omitempty"`
}

// HeaderPolicy is the action to take on stateful headers
type HeaderPolicy string

const (
	// HeaderPolicyEnsure headers, will create or copy stateful headers to the new CloudEvent.
	HeaderPolicyEnsure HeaderPolicy = "ensure"
	// HeaderPolicyPropagate will copy stateful headers to the new CloudEvent.
	HeaderPolicyPropagate HeaderPolicy = "propagate"
	// HeaderPolicyNone wont copy stateful headers to the new CloudEvent.
	HeaderPolicyNone HeaderPolicy = "none"
)

// InfraTargetState holds the state options
type InfraTargetState struct {
	// HeadersPolicy determines actions on stateful headers.
	HeadersPolicy *HeaderPolicy `json:"headersPolicy,omitempty"`

	// Bridge is the identifier to be used if the adapter needs to
	// create cloud events headers as part of its policy.
	//
	// The Bridge moniker identifies uniquely the workflow that
	// this component is part of, and should be taken into account
	// when storing variables in the state store.
	Bridge *string `json:"bridge,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfraTargetList is a list of InfraTarget resources
type InfraTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []InfraTarget `json:"items"`
}
