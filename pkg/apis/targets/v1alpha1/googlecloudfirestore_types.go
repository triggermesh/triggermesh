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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudFirestoreTarget is the Schema for the GoogleCloudFirestore Target.
type GoogleCloudFirestoreTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudFirestoreTargetSpec `json:"spec"`
	Status v1alpha1.Status                `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*GoogleCloudFirestoreTarget)(nil)
	_ v1alpha1.AdapterConfigurable    = (*GoogleCloudFirestoreTarget)(nil)
	_ v1alpha1.EventReceiver          = (*GoogleCloudFirestoreTarget)(nil)
	_ v1alpha1.EventSource            = (*GoogleCloudFirestoreTarget)(nil)
	_ v1alpha1.ServiceAccountProvider = (*GoogleCloudFirestoreTarget)(nil)
)

// GoogleCloudFirestoreTargetSpec defines the desired state of the event target.
type GoogleCloudFirestoreTargetSpec struct {
	// Credentials represents how Google Firestore credentials should be provided in the secret
	// Deprecated, please use "auth" object.
	Credentials *SecretValueFromSource `json:"credentialsJson,omitempty"`

	// Authentication methods common for all GCP targets.
	Auth *v1alpha1.GoogleCloudAuth `json:"auth,omitempty"`

	// DefaultCollection sets a default Firestore collection to select from
	DefaultCollection string `json:"defaultCollection"`

	// ProjectID specifies the Google project ID
	ProjectID string `json:"projectID"`

	// Whether to omit CloudEvent context attributes in documents created in Firestore.
	// When this property is false (default), the entire CloudEvent payload is included.
	// When this property is true, only the CloudEvent data is included.
	DiscardCEContext bool `json:"discardCloudEventContext"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudFirestoreTargetList is a list of event target instances.
type GoogleCloudFirestoreTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GoogleCloudFirestoreTarget `json:"items"`
}
