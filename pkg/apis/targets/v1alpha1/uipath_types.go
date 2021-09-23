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

// UiPathTarget is the Schema for the event target.
type UiPathTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UiPathTargetSpec   `json:"spec,omitempty"`
	Status UiPathTargetStatus `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ runtime.Object            = (*UiPathTarget)(nil)
	_ kmeta.OwnerRefable        = (*UiPathTarget)(nil)
	_ targets.IntegrationTarget = (*UiPathTarget)(nil)
	_ targets.EventSource       = (*UiPathTarget)(nil)
	_ duckv1.KRShaped           = (*UiPathTarget)(nil)
)

// UiPathTargetSpec defines the desired state of the event target.
type UiPathTargetSpec struct {
	UserKey            *SecretValueFromSource `json:"userKey"`
	RobotName          string                 `json:"robotName"`
	ProcessName        string                 `json:"processName"`
	TenantName         string                 `json:"tenantName"`
	AccountLogicalName string                 `json:"accountLogicalName"`
	ClientID           string                 `json:"clientID"`
	OrganizationUnitID string                 `json:"organizationUnitID"`
}

// UiPathTargetStatus defines the observed state of the event target.
type UiPathTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`

	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UiPathTargetList contains a list of event targets.
type UiPathTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UiPathTarget `json:"items"`
}
