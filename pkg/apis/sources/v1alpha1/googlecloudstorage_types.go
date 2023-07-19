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

// GoogleCloudStorageSource is the Schema for the event source.
type GoogleCloudStorageSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudStorageSourceSpec   `json:"spec,omitempty"`
	Status GoogleCloudStorageSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*GoogleCloudStorageSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*GoogleCloudStorageSource)(nil)
	_ v1alpha1.EventSource            = (*GoogleCloudStorageSource)(nil)
	_ v1alpha1.EventSender            = (*GoogleCloudStorageSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*GoogleCloudStorageSource)(nil)
)

// GoogleCloudStorageSourceSpec defines the desired state of the event source.
type GoogleCloudStorageSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Name of the Cloud Storage bucket to receive change notifications from.
	Bucket string `json:"bucket"`

	// Settings related to the Pub/Sub resources associated with the bucket.
	PubSub GoogleCloudSourcePubSubSpec `json:"pubsub"`

	// Types of events to subscribe to.
	//
	// The list of available event types can be found at
	// https://cloud.google.com/storage/docs/pubsub-notifications#events
	//
	// All types are selected when this attribute is not set.
	//
	// +optional
	EventTypes []string `json:"eventTypes,omitempty"`

	// Object name prefix filter
	//
	// If present, will only receive notifications for objects whose names that begin with this prefix.
	//
	// If not set, notifications are received for all objects.
	//
	// +optional
	ObjectNamePrefix string `json:"objectNamePrefix,omitempty"`

	// Different authentication methods available in sources on GCP.
	Auth v1alpha1.GoogleCloudAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// GoogleCloudStorageSourceStatus defines the observed state of the event source.
type GoogleCloudStorageSourceStatus struct {
	v1alpha1.Status `json:",inline"`

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
