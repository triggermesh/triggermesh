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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudBillingSource is the Schema for the event source.
type GoogleCloudBillingSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudBillingSourceSpec   `json:"spec,omitempty"`
	Status GoogleCloudBillingSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*GoogleCloudBillingSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*GoogleCloudBillingSource)(nil)
	_ v1alpha1.EventSource            = (*GoogleCloudBillingSource)(nil)
	_ v1alpha1.EventSender            = (*GoogleCloudBillingSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*GoogleCloudBillingSource)(nil)
)

// GoogleCloudBillingSourceSpec defines the desired state of the event source.
type GoogleCloudBillingSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// The identifier for the Cloud Billing account owning the budget.
	BillingAccountID string `json:"billingAccountId"`

	// The identifier for the Cloud Billing budget.
	// You can locate the budget's ID in your budget under Manage notifications.
	// The ID is displayed after you select Connect a Pub/Sub topic to this budget.
	BudgetID string `json:"budgetId"`

	// Settings related to the Pub/Sub resources associated with the Billing budget event sink.
	PubSub GoogleCloudSourcePubSubSpec `json:"pubsub"`

	// Different authentication methods available in sources on GCP.
	Auth v1alpha1.GoogleCloudAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// GoogleCloudBillingSourceStatus defines the observed state of the event source.
type GoogleCloudBillingSourceStatus struct {
	v1alpha1.Status `json:",inline"`

	// Resource name of the target Pub/Sub topic.
	Topic *GCloudResourceName `json:"topic,omitempty"`

	// Resource name of the managed Pub/Sub subscription associated with
	// the managed topic.
	Subscription *GCloudResourceName `json:"subscription,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudBillingSourceList contains a list of event sources.
type GoogleCloudBillingSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GoogleCloudBillingSource `json:"items"`
}
