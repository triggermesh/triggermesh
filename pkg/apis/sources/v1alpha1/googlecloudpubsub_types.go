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

// GoogleCloudPubSubSource is the Schema for the event source.
type GoogleCloudPubSubSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudPubSubSourceSpec   `json:"spec,omitempty"`
	Status GoogleCloudPubSubSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*GoogleCloudPubSubSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*GoogleCloudPubSubSource)(nil)
	_ v1alpha1.EventSource            = (*GoogleCloudPubSubSource)(nil)
	_ v1alpha1.EventSender            = (*GoogleCloudPubSubSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*GoogleCloudPubSubSource)(nil)
)

// GoogleCloudPubSubSourceSpec defines the desired state of the event source.
type GoogleCloudPubSubSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Full resource name of the Pub/Sub topic to subscribe to, in the
	// format "projects/{project_name}/topics/{topic_name}".
	Topic GCloudResourceName `json:"topic"`

	// ID of the subscription to use to pull messages from the topic.
	//
	// If supplied, this subscription must 1) exist and 2) belong to the
	// provided topic. Otherwise, a pull subscription to that topic is
	// created on behalf of the user.
	//
	// +optional
	SubscriptionID *string `json:"subscriptionID,omitempty"`

	// Different authentication methods available in sources on GCP.
	Auth v1alpha1.GoogleCloudAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// GoogleCloudPubSubSourceStatus defines the observed state of the event source.
type GoogleCloudPubSubSourceStatus struct {
	v1alpha1.Status `json:",inline"`
	Subscription    *GCloudResourceName `json:"subscription,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudPubSubSourceList contains a list of event sources.
type GoogleCloudPubSubSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GoogleCloudPubSubSource `json:"items"`
}
