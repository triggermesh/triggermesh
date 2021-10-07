/*
Copyright 2021 TriggerMesh Inc.

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
	"k8s.io/apimachinery/pkg/runtime"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SlackTarget defines the schema for the Slack target.
type SlackTarget struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the SlackTarget (from the client).
	Spec SlackTargetSpec `json:"spec"`

	// Status communicates the observed state of the SlackTarget (from the controller).
	// +optional
	Status SlackTargetStatus `json:"status,omitempty"`
}

// SlackTargetSpec defines the spec for the Slack Taret.
type SlackTargetSpec struct {
	// Token for Slack App
	Token SecretValueFromSource `json:"token"`
}

// Check the interfaces SlackTarget should be implementing.
var (
	_ runtime.Object            = (*SlackTarget)(nil)
	_ kmeta.OwnerRefable        = (*SlackTarget)(nil)
	_ targets.IntegrationTarget = (*SlackTarget)(nil)
	_ targets.EventSource       = (*SlackTarget)(nil)
	_ duckv1.KRShaped           = (*SlackTarget)(nil)
)

// SlackTargetStatus communicates the observed state of the SlackTarget (from the controller).
type SlackTargetStatus struct {
	// inherits duck/v1beta1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last
	//   processed by the controller.
	// * Conditions - the latest available observations of a resource's current
	//   state.
	duckv1.Status `json:",inline"`

	// AddressStatus fulfills the Addressable contract.
	duckv1.AddressStatus `json:",inline"`

	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SlackTargetList is a list of event targets.
type SlackTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []SlackTarget `json:"items"`
}
