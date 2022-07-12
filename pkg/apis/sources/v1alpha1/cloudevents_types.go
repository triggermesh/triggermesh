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

// CloudEventsSource is the Schema for the event source.
type CloudEventsSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudEventsSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status       `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*CloudEventsSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*CloudEventsSource)(nil)
	_ v1alpha1.EventSource         = (*CloudEventsSource)(nil)
	_ v1alpha1.EventSender         = (*CloudEventsSource)(nil)
)

// CloudEventsSourceSpec defines the desired state of the event source.
type CloudEventsSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Credentials to connect to this source.
	// +optional
	Credentials *HTTPCredentials `json:"credentials,omitempty"`

	// Path under which requests are accepted.
	// +optional
	Path *string `json:"path,omitempty"`

	// RateLimiter for incoming events per adapter instance.
	// A single CloudEventsSource object can create multiple adapter instances,
	// the rate limiting configuration being applied to each of them individually.
	// +optional
	RateLimiter *RateLimiter `json:"rateLimiter,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// HTTPCredentials to be used when receiving requests.
type HTTPCredentials struct {
	BasicAuths []HTTPBasicAuth `json:"basicAuths,omitempty"`
}

// HTTPBasicAuth credentials.
type HTTPBasicAuth struct {
	Username string                  `json:"username"`
	Password v1alpha1.ValueFromField `json:"password"`
}

// RateLimiter parameters.
type RateLimiter struct {
	// RequestsPerSecond is used to limit the number of requests that a
	// single instance of the CloudEventsSource adapter can accept.
	RequestsPerSecond int `json:"requestsPerSecond"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudEventsSourceList contains a list of event sources.
type CloudEventsSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudEventsSource `json:"items"`
}
