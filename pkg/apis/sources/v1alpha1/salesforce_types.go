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
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SalesforceSource is the Schema for the event source.
type SalesforceSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SalesforceSourceSpec `json:"spec,omitempty"`
	Status EventSourceStatus    `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ Reconcilable = (*SalesforceSource)(nil)
)

// SalesforceSourceSpec defines the desired state of the event source.
type SalesforceSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Authentication method to interact with the Salesforce API.
	Auth SalesforceAuth `json:"auth"`

	// APIVersion at Salesforce.
	// +optional
	APIVersion *string `json:"apiVersion"`

	// Subscription to a Salesforce channel
	Subscription SalesforceSubscription `json:"subscription"`
}

// SalesforceSubscription to connect to.
type SalesforceSubscription struct {
	Channel  string `json:"channel"`
	ReplayID *int   `json:"replayID,omitempty"`
}

// SalesforceAuth contains Salesforce credentials.
type SalesforceAuth struct {
	ClientID string         `json:"clientID"`
	Server   string         `json:"server"`
	User     string         `json:"user"`
	CertKey  ValueFromField `json:"certKey"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SalesforceSourceList contains a list of event sources.
type SalesforceSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SalesforceSource `json:"items"`
}
