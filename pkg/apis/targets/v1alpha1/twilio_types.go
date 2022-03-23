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

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TwilioTarget is the Schema for an Twilio Target.
type TwilioTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TwilioTargetSpec `json:"spec"`
	Status TargetStatus     `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ Reconcilable              = (*TwilioTarget)(nil)
	_ targets.IntegrationTarget = (*TwilioTarget)(nil)
	_ targets.EventSource       = (*TwilioTarget)(nil)
)

// TwilioTargetSpec holds the desired state of the TwilioTarget.
type TwilioTargetSpec struct {

	// Twilio account SID
	AccountSID SecretValueFromSource `json:"sid"`

	// Twilio account Token
	Token SecretValueFromSource `json:"token"`

	// DefaultPhoneFrom is the purchased Twilio phone we are using
	// +optional
	DefaultPhoneFrom *string `json:"defaultPhoneFrom,omitempty"`

	// DefaultPhoneTo is the destination phone
	// +optional
	DefaultPhoneTo *string `json:"defaultPhoneTo,omitempty"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TwilioTargetList is a list of TwilioTarget resources
type TwilioTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TwilioTarget `json:"items"`
}
