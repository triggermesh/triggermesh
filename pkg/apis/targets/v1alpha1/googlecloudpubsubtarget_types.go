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
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudPubSubTarget is the Schema the event target.
type GoogleCloudPubSubTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudPubSubTargetSpec `json:"spec"`
	Status v1alpha1.Status             `json:"status,omitempty"`
}

// Check the interfaces GoogleCloudPubSubTarget should be implementing.
var (
	_ runtime.Object                  = (*GoogleCloudPubSubTarget)(nil)
	_ kmeta.OwnerRefable              = (*GoogleCloudPubSubTarget)(nil)
	_ duckv1.KRShaped                 = (*GoogleCloudPubSubTarget)(nil)
	_ v1alpha1.ServiceAccountProvider = (*GoogleCloudPubSubTarget)(nil)
)

// GoogleCloudPubSubTargetSpec holds the desired state of the event target.
type GoogleCloudPubSubTargetSpec struct {
	// Full resource name of the Pub/Sub topic to subscribe to, in the
	// format "projects/{project_name}/topics/{topic_name}".
	Topic GCloudResourceName `json:"topic"`

	// Service account key in JSON format.
	// https://cloud.google.com/iam/docs/creating-managing-service-account-keys
	// Deprecated, please use "auth" object.
	ServiceAccountKey *SecretValueFromSource `json:"credentialsJson,omitempty"`

	// Authentication methods common for all GCP targets.
	Auth *v1alpha1.GoogleCloudAuth `json:"auth,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`

	// DiscardCloudEventContext is the policy for how to handle the payload of
	// the CloudEvent.
	DiscardCloudEventContext bool `json:"discardCloudEventContext,omitempty"`
}

// GoogleCloudPubSubTargetStatus communicates the observed state of the event target.
type GoogleCloudPubSubTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudPubSubTargetList is a list of event target instances.
type GoogleCloudPubSubTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GoogleCloudPubSubTarget `json:"items"`
}
