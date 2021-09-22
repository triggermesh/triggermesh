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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SalesforceTarget is the Schema for the Salesforce Target.
type SalesforceTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SalesforceTargetSpec   `json:"spec"`
	Status SalesforceTargetStatus `json:"status,omitempty"`
}

// Check the interfaces SalesforceTarget should be implementing.
var (
	_ runtime.Object            = (*SalesforceTarget)(nil)
	_ kmeta.OwnerRefable        = (*SalesforceTarget)(nil)
	_ targets.IntegrationTarget = (*SalesforceTarget)(nil)
	_ targets.EventSource       = (*SalesforceTarget)(nil)
	_ duckv1.KRShaped           = (*SalesforceTarget)(nil)
)

// SalesforceTargetSpec holds the desired state of the SalesforceTarget.
type SalesforceTargetSpec struct {
	// Authentication method to interact with the Salesforce API.
	Auth SalesforceAuth `json:"auth"`

	// APIVersion at Salesforce.
	// +optional
	APIVersion *string `json:"apiVersion"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// SalesforceAuth contains Salesforce credentials.
type SalesforceAuth struct {
	ClientID string                `json:"clientID"`
	Server   string                `json:"server"`
	User     string                `json:"user"`
	CertKey  SecretValueFromSource `json:"certKey"`
}

// SalesforceTargetStatus communicates the observed state of the SalesforceTarget (from the controller).
type SalesforceTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`
	CloudEventStatus     `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SalesforceTargetList is a list of SalesforceTarget resources
type SalesforceTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []SalesforceTarget `json:"items"`
}
