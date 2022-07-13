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
	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureSentinelTarget is the Schema for an Azure Sentinel Target.
type AzureSentinelTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureSentinelTargetSpec `json:"spec,omitempty"`
	Status v1alpha1.Status         `json:"status,omitempty"`
}

// Check the interfaces AzureSentinelTarget should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AzureSentinelTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*AzureSentinelTarget)(nil)
	_ v1alpha1.EventReceiver       = (*AzureSentinelTarget)(nil)
	_ v1alpha1.EventSource         = (*AzureSentinelTarget)(nil)
)

// AzureSentinelTargetSpec holds the desired state of the event target.
type AzureSentinelTargetSpec struct {
	// SubscriptionID refers to the Azure Subscription ID that the Azure Sentinel instance is associated with.
	SubscriptionID string `json:"subscriptionID"`
	// ResourceGroup refers to the resource group where the Azure Sentinel instance is deployed.
	ResourceGroup string `json:"resourceGroup"`
	// Workspace refers to the workspace name in Azure Sentinel.
	Workspace string `json:"workspace"`
	// ClientID refers to the Application (client) ID of the App Registration. see -> https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app for more details
	ClientID string `json:"clientID"`
	// ClientSecret refers to the Client Secret of an App Registration. see -> https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app for more details
	ClientSecret SecretValueFromSource `json:"clientSecret"`
	// TenantID refers to the Directory (tenant) ID of the App Registration. see -> https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app for more details
	TenantID string `json:"tenantID"`
	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}


// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureSentinelTargetList is a list of event target instances.
type AzureSentinelTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AzureSentinelTarget `json:"items"`
}
