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

	pkgapis "knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis"
	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HTTPPollerSource is the schema for the event source.
type HTTPPollerSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HTTPPollerSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status      `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*HTTPPollerSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*HTTPPollerSource)(nil)
	_ v1alpha1.EventSource         = (*HTTPPollerSource)(nil)
	_ v1alpha1.EventSender         = (*HTTPPollerSource)(nil)
)

// HTTPPollerSourceSpec defines the desired state of the event source.
type HTTPPollerSourceSpec struct {
	// inherits duck/v1 SourceSpec, which currently provides:
	// * Sink - a reference to an object that will resolve to a domain name or
	//   a URI directly to use as the sink.
	// * CloudEventOverrides - defines overrides to control the output format
	//   and modifications of the event sent to the sink.
	duckv1.SourceSpec `json:",inline"`

	// Value of the CloudEvents 'type' attribute to set on ingested events.
	// https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#type
	EventType string `json:"eventType"`

	// Value of the CloudEvents 'source' attribute to set on ingested events.
	// https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#source-1
	// +optional
	EventSource *string `json:"eventSource,omitempty"`

	// HTTP/S URL of the endpoint to poll data from.
	Endpoint pkgapis.URL `json:"endpoint"`

	// HTTP request method to use in requests to the specified 'endpoint'.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods
	Method string `json:"method"`

	// Controls whether the HTTP client verifies the server's certificate
	// chain and host name when communicating over TLS.
	// +optional
	SkipVerify *bool `json:"skipVerify,omitempty"`

	// CA certificate in X.509 format the HTTP client should use to verify
	// the identity of remote servers when communicating over TLS.
	// +optional
	CACertificate *string `json:"caCertificate,omitempty"`

	// User name to set in HTTP requests that require HTTP Basic authentication.
	// +optional
	BasicAuthUsername *string `json:"basicAuthUsername,omitempty"`

	// Password to set in HTTP requests that require HTTP Basic authentication.
	// +optional
	BasicAuthPassword *v1alpha1.ValueFromField `json:"basicAuthPassword,omitempty"`

	// HTTP headers to include in HTTP requests.
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// Duration which defines how often the HTTP/S endpoint should be polled.
	// Expressed as a duration string, which format is documented at https://pkg.go.dev/time#ParseDuration.
	Interval apis.Duration `json:"interval"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HTTPPollerSourceList contains a list of event sources.
type HTTPPollerSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HTTPPollerSource `json:"items"`
}
