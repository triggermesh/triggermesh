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
	_ v1alpha1.Reconcilable = (*GoogleCloudSourceRepositoriesSource)(nil)
	_ v1alpha1.EventSource  = (*GoogleCloudSourceRepositoriesSource)(nil)
	_ v1alpha1.EventSender  = (*GoogleCloudSourceRepositoriesSource)(nil)
)

// GoogleCloudSourceRepositoriesSourceSpec defines the desired state of the event source.
type GoogleCloudSourceRepositoriesSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Name of the Cloud repo to receive notifications from.
	Repository GCloudResourceName `json:"repository"`

	// Settings related to the Pub/Sub resources associated with the repo events.
	PubSub GoogleCloudSourceRepositoriesSourcePubSubSpec `json:"pubsub"`

	// Service account key in JSON format.
	// https://cloud.google.com/iam/docs/creating-managing-service-account-keys
	ServiceAccountKey v1alpha1.ValueFromField `json:"serviceAccountKey"`
}

// GoogleCloudSourceRepositoriesSourcePubSubSpec defines the attributes related to the
// configuration of Pub/Sub resources.
type GoogleCloudSourceRepositoriesSourcePubSubSpec struct {
	// Optional: no more than one of the following may be specified.

	// Full resource name of the Pub/Sub topic where change notifications
	// originating from the configured sink are sent to. If not supplied,
	// a topic is created on behalf of the user, in the GCP project
	// referenced by the Project attribute.
	//
	// The expected format is described at https://cloud.google.com/pubsub/docs/admin#resource_names:
	//   "projects/{project_name}/topics/{topic_name}"
	//
	// +optional
	Topic *GCloudResourceName `json:"topic,omitempty"`

	// Name of the GCP project where Pub/Sub resources associated with the
	// Cloud repo are to be created.
	//
	// Mutually exclusive with Topic which, if supplied, already contains
	// the project name.
	//
	// +optional
	Project *string `json:"project,omitempty"`
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
