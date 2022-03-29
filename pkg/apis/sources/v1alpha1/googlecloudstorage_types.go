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

// GoogleCloudStorageSource is the Schema for the event source.
type GoogleCloudStorageSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudStorageSourceSpec   `json:"spec,omitempty"`
	Status GoogleCloudStorageSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ Reconcilable = (*GoogleCloudStorageSource)(nil)
)

// GoogleCloudStorageSourceSpec defines the desired state of the event source.
type GoogleCloudStorageSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Name of the Cloud Storage bucket to receive change notifications from.
	Bucket string `json:"bucket"`

	// Settings related to the Pub/Sub resources associated with the bucket.
	PubSub GoogleCloudStorageSourcePubSubSpec `json:"pubsub"`

	// Types of events to subscribe to.
	//
	// The list of available event types can be found at
	// https://cloud.google.com/storage/docs/pubsub-notifications#events
	//
	// All types are selected when this attribute is not set.
	//
	// +optional
	EventTypes []string `json:"eventTypes,omitempty"`

	// Service account key in JSON format.
	// https://cloud.google.com/iam/docs/creating-managing-service-account-keys
	ServiceAccountKey ValueFromField `json:"serviceAccountKey"`
}

// GoogleCloudStorageSourcePubSubSpec defines the attributes related to the
// configuration of Pub/Sub resources.
type GoogleCloudStorageSourcePubSubSpec struct {
	// Optional: no more than one of the following may be specified.

	// Full resource name of the Pub/Sub topic where change notifications
	// originating from the configured bucket are sent to. If not supplied,
	// a topic is created on behalf of the user, in the GCP project
	// referenced by the Project attribute.
	//
	// The expected format is described at https://cloud.google.com/pubsub/docs/admin#resource_names:
	//   "projects/{project_name}/topics/{topic_name}"
	//
	// +optional
	Topic *GCloudResourceName `json:"topic,omitempty"`

	// Name of the GCP project where Pub/Sub resources associated with the
	// Cloud Storage bucket are to be created.
	//
	// Mutually exclusive with Topic which, if supplied, already contains
	// the project name.
	//
	// +optional
	Project *string `json:"project,omitempty"`
}

// GoogleCloudStorageSourceStatus defines the observed state of the event source.
type GoogleCloudStorageSourceStatus struct {
	EventSourceStatus `json:",inline"`

	// ID of the managed Cloud Storage bucket notification configuration.
	NotificationID *string `json:"notificationID,omitempty"`

	// Resource name of the target Pub/Sub topic.
	Topic *GCloudResourceName `json:"topic,omitempty"`
	// Resource name of the managed Pub/Sub subscription associated with
	// the managed topic.
	Subscription *GCloudResourceName `json:"subscription,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudStorageSourceList contains a list of event sources.
type GoogleCloudStorageSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GoogleCloudStorageSource `json:"items"`
}
