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

// JiraTarget is the Schema for the Infra JS Target.
type JiraTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JiraTargetSpec   `json:"spec"`
	Status JiraTargetStatus `json:"status,omitempty"`
}

// Check the interfaces JiraTarget should be implementing.
var (
	_ runtime.Object     = (*JiraTarget)(nil)
	_ kmeta.OwnerRefable = (*JiraTarget)(nil)
	_ duckv1.KRShaped    = (*JiraTarget)(nil)
)

// JiraTargetSpec holds the desired state of the JiraTarget.
type JiraTargetSpec struct {
	// Authentication to interact with the Salesforce API.
	Auth JiraAuth `json:"auth"`

	// URL for Jira service
	URL string `json:"url"`
}

// JiraAuth contains Jira credentials.
type JiraAuth struct {
	User  string                `json:"user"`
	Token SecretValueFromSource `json:"token"`
}

// JiraTargetStatus communicates the observed state of the JiraTarget (from the controller).
type JiraTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JiraTargetList is a list of JiraTarget resources
type JiraTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []JiraTarget `json:"items"`
}
