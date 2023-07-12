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

// GoogleCloudSourceRepositoriesSource is the Schema for the event source.
type GoogleCloudSourceRepositoriesSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudSourceRepositoriesSourceSpec   `json:"spec,omitempty"`
	Status GoogleCloudSourceRepositoriesSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*GoogleCloudSourceRepositoriesSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*GoogleCloudSourceRepositoriesSource)(nil)
	_ v1alpha1.EventSource            = (*GoogleCloudSourceRepositoriesSource)(nil)
	_ v1alpha1.EventSender            = (*GoogleCloudSourceRepositoriesSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*GoogleCloudSourceRepositoriesSource)(nil)
)

// GoogleCloudSourceRepositoriesSourceSpec defines the desired state of the event source.
type GoogleCloudSourceRepositoriesSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Name of the Cloud repo to receive notifications from.
	Repository GCloudResourceName `json:"repository"`

	// Settings related to the Pub/Sub resources associated with the repo events.
	PubSub GoogleCloudSourcePubSubSpec `json:"pubsub"`

	// Email address of the service account used for publishing
	// notifications to Pub/Sub. This service account needs to be in the
	// same project as the repo, and to have the 'pubsub.topics.publish'
	// IAM permission associated with it. It can (but doesn't have to) be
	// the same service account as the 'ServiceAccountKey' attribute.
	//
	// If unspecified, it defaults to the Compute Engine default service
	// account.
	//
	// +optional
	PublishServiceAccount *string `json:"publishServiceAccount,omitempty"`

	// Different authentication methods available in sources on GCP.
	Auth v1alpha1.GoogleCloudAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// GoogleCloudSourceRepositoriesSourceStatus defines the observed state of the event source.
type GoogleCloudSourceRepositoriesSourceStatus struct {
	v1alpha1.Status `json:",inline"`

	// Resource name of the target Pub/Sub topic.
	Topic *GCloudResourceName `json:"topic,omitempty"`

	// Resource name of the managed Pub/Sub subscription associated with
	// the managed topic.
	Subscription *GCloudResourceName `json:"subscription,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudSourceRepositoriesSourceList contains a list of event sources.
type GoogleCloudSourceRepositoriesSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GoogleCloudSourceRepositoriesSource `json:"items"`
}
