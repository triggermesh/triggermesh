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

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SendGridTarget is the Schema for an Sendgrid Target.
type SendGridTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SendGridTargetSpec `json:"spec"`
	Status TargetStatus       `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ Reconcilable              = (*SendGridTarget)(nil)
	_ targets.IntegrationTarget = (*SendGridTarget)(nil)
	_ targets.EventSource       = (*SendGridTarget)(nil)
)

// SendGridTargetSpec holds the desired state of the SendGridTarget.
type SendGridTargetSpec struct {

	// APIKey for account
	APIKey SecretValueFromSource `json:"apiKey"`

	// DefaultFromEmail is a default email account to assign to the outgoing email's.
	// +optional
	DefaultFromEmail *string `json:"defaultFromEmail,omitempty"`

	// DefaultToEmail is a default recipient email account to assign to the outgoing email's.
	// +optional
	DefaultToEmail *string `json:"defaultToEmail,omitempty"`

	// DefaultToName is a default recipient name to assign to the outgoing email's.
	// +optional
	DefaultToName *string `json:"defaultToName,omitempty"`

	// DefaultFromName is a default sender name to assign to the outgoing email's.
	// +optional
	DefaultFromName *string `json:"defaultFromName,omitempty"`

	// DefaultMessage is a default message to assign to the outgoing email's.
	// +optional
	DefaultMessage *string `json:"defaultMessage,omitempty"`

	// DefaultSubject is a default subject to assign to the outgoing email's.
	// +optional
	DefaultSubject *string `json:"defaultSubject,omitempty"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SendGridTargetList is a list of SendGridTarget resources
type SendGridTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []SendGridTarget `json:"items"`
}
