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

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UiPathTarget is the Schema for the event target.
type UiPathTarget struct { //nolint:stylecheck
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UiPathTargetSpec `json:"spec,omitempty"`
	Status TargetStatus     `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ Reconcilable              = (*UiPathTarget)(nil)
	_ targets.IntegrationTarget = (*UiPathTarget)(nil)
	_ targets.EventSource       = (*UiPathTarget)(nil)
)

// UiPathTargetSpec defines the desired state of the event target.
type UiPathTargetSpec struct { //nolint:stylecheck
	// UserKey An OAuth token used to obtain an access key.
	UserKey *SecretValueFromSource `json:"userKey"`
	// RobotName is the robot to invoke with this target.
	RobotName string `json:"robotName"`
	// ProccessName is the process name that will be used by UiPath for the target.
	ProcessName string `json:"processName"`
	// TenantName is the tenant that contains the components that will be invoked by the target.
	TenantName string `json:"tenantName"`
	// AccountLogicalName is the unique site URL used to identif the UiPath tenant.
	AccountLogicalName string `json:"accountLogicalName"`
	// ClientID is the OAuth id registered to this target.
	ClientID string `json:"clientID"`
	// OrganizationUnitID is the organization unit within the tenant that the UiPath proccess will run under.
	OrganizationUnitID string `json:"organizationUnitID"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UiPathTargetList contains a list of event targets.
type UiPathTargetList struct { //nolint:stylecheck
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UiPathTarget `json:"items"`
}
