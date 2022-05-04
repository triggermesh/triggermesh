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
	"knative.dev/pkg/apis"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudEventsTarget is a gateway that produces received CloudEvents to a destination.
type CloudEventsTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudEventsTargetSpec `json:"spec,omitempty"`
	Status v1alpha1.Status       `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*CloudEventsTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*CloudEventsTarget)(nil)
)

// CloudEventsTargetSpec defines the desired state of the event target.
type CloudEventsTargetSpec struct {
	// Credentials to connect to the remote endpoint.
	// +optional
	Credentials *CloudEventsCredentials `json:"credentials,omitempty"`

	// Path at the remote endpoint under which requests are accepted.
	// +optional
	Path *string `json:"path,omitempty"`

	// Endpoint that accept CloudEvents.
	Endpoint apis.URL `json:"endpoint"`

	// AdapterOverrides sets runtime parameters to the adapter instance.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// CloudEventsCredentials to be used when sending requests.
type CloudEventsCredentials struct {
	BasicAuth HTTPBasicAuth `json:"basicAuth,omitempty"`
}

// HTTPBasicAuth credentials.
type HTTPBasicAuth struct {
	Username string                  `json:"username"`
	Password v1alpha1.ValueFromField `json:"password"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudEventsTargetList is a list of event target instances.
type CloudEventsTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []CloudEventsTarget `json:"items"`
}
