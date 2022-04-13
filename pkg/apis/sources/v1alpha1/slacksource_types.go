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

// SlackSource is the schema for the event source.
type SlackSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SlackSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*SlackSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*SlackSource)(nil)
	_ v1alpha1.EventSource         = (*SlackSource)(nil)
	_ v1alpha1.EventSender         = (*SlackSource)(nil)
)

// SlackSourceSpec defines the desired state of the event source.
type SlackSourceSpec struct {
	// inherits duck/v1 SourceSpec, which currently provides:
	// * Sink - a reference to an object that will resolve to a domain name or
	//   a URI directly to use as the sink.
	// * CloudEventOverrides - defines overrides to control the output format
	//   and modifications of the event sent to the sink.
	duckv1.SourceSpec `json:",inline"`

	// SigningSecret can be set to the value of Slack request signing secret
	// to authenticate callbacks.
	// See: https://api.slack.com/authentication/verifying-requests-from-slack
	// +optional
	SigningSecret *v1alpha1.ValueFromField `json:"signingSecret,omitempty"`

	// AppID identifies the Slack application generating this event.
	// It helps identifying the App sourcing events when multiple Slack
	// applications shared an endpoint. See: https://api.slack.com/events-api
	// +optional
	AppID *string `json:"appID,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SlackSourceList contains a list of event sources.
type SlackSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SlackSource `json:"items"`
}
