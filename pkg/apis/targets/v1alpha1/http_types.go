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

// HTTPTarget is the Schema for an HTTP Target.
type HTTPTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HTTPTargetSpec  `json:"spec"`
	Status v1alpha1.Status `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.EventSource         = (*HTTPTarget)(nil)
	_ v1alpha1.EventReceiver       = (*HTTPTarget)(nil)
	_ v1alpha1.Reconcilable        = (*HTTPTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*HTTPTarget)(nil)
)

// HTTPTargetSpec defines the desired state of the event target.
type HTTPTargetSpec struct {
	// Response data to be used at replies.
	Response HTTPEventResponse `json:"response"`

	// Endpoint to connect to.
	Endpoint apis.URL `json:"endpoint"`

	// Method to use at requests.
	Method string `json:"method"`

	// Headers to be included at HTTP requests
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// SkipVerify disables server certificate validation.
	// +optional
	SkipVerify *bool `json:"skipVerify"`

	// CACertificate uses the CA certificate to verify the remote server certificate.
	// +optional
	CACertificate *string `json:"caCertificate"`

	// BasicAuthUsername used for basic authentication.
	// +optional
	BasicAuthUsername *string `json:"basicAuthUsername,omitempty"`

	// BasicAuthPassword used for basic authentication.
	// +optional
	BasicAuthPassword SecretValueFromSource `json:"basicAuthPassword,omitempty"`

	// OAuthClientID used for OAuth2 authentication.
	// +optional
	OAuthClientID *string `json:"oauthClientID,omitempty"`

	// OAuthClientSecret used for OAuth2 authentication.
	// +optional
	OAuthClientSecret SecretValueFromSource `json:"oauthClientSecret,omitempty"`

	// OAuthTokenURL used for OAuth2 authentication.
	// +optional
	OAuthTokenURL *string `json:"oauthTokenURL,omitempty"`

	// OAuthScopes used for OAuth2 authentication.
	// +optional
	OAuthScopes *[]string `json:"oauthScopes,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// HTTPEventResponse for reply events context.
type HTTPEventResponse struct {
	// EventType for the reply.
	EventType string `json:"eventType"`

	// EventSource for the reply.
	EventSource string `json:"eventSource"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HTTPTargetList is a list of event target instances.
type HTTPTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []HTTPTarget `json:"items"`
}
