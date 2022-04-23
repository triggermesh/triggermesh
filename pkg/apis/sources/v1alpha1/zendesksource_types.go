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

// ZendeskSource is the schema for the event source.
type ZendeskSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZendeskSourceSpec   `json:"spec,omitempty"`
	Status ZendeskSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*ZendeskSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*ZendeskSource)(nil)
	_ v1alpha1.MultiTenant         = (*ZendeskSource)(nil)
	_ v1alpha1.EventSource         = (*ZendeskSource)(nil)
	_ v1alpha1.EventSender         = (*ZendeskSource)(nil)
)

// ZendeskSourceSpec defines the desired state of the event source.
type ZendeskSourceSpec struct {
	// inherits duck/v1 SourceSpec, which currently provides:
	// * Sink - a reference to an object that will resolve to a domain name or
	//   a URI directly to use as the sink.
	// * CloudEventOverrides - defines overrides to control the output format
	//   and modifications of the event sent to the sink.
	duckv1.SourceSpec `json:",inline"`

	// Token identifies the API token used for creating the proper credentials to interface with Zendesk
	// allowing the source to auto-register the webhook to authenticate callbacks.
	Token v1alpha1.ValueFromField `json:"token,omitempty"`

	// Email identifies the email used for creating the proper credentials to interface with Zendesk
	// allowing the source to auto-register the webhook to authenticate callbacks.
	Email string `json:"email,omitempty"`

	// WebhookPassword used for basic authentication for events sent from Zendesk
	// to the adapter.
	WebhookPassword v1alpha1.ValueFromField `json:"webhookPassword,omitempty"`

	// WebhookUsername used for basic authentication for events sent from Zendesk
	// to the adapter.
	WebhookUsername string `json:"webhookUsername,omitempty"`

	// Subdomain identifies Zendesk subdomain
	Subdomain string `json:"subdomain,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// ZendeskSourceStatus defines the observed state of the event source.
type ZendeskSourceStatus struct {
	v1alpha1.Status `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ZendeskSourceList contains a list of event sources.
type ZendeskSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZendeskSource `json:"items"`
}
