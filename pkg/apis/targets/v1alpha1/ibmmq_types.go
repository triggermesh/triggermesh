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

// IBMMQTarget is the Schema the event target.
type IBMMQTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IBMMQTargetSpec `json:"spec"`
	Status v1alpha1.Status `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*IBMMQTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*IBMMQTarget)(nil)
	_ v1alpha1.EventReceiver       = (*IBMMQTarget)(nil)
	_ v1alpha1.EventSource         = (*IBMMQTarget)(nil)
)

// IBMMQTargetSpec defines the desired state of the event target.
type IBMMQTargetSpec struct {
	ConnectionName string `json:"connectionName"`
	QueueManager   string `json:"queueManager"`
	QueueName      string `json:"queueName"`
	ChannelName    string `json:"channelName"`

	ReplyTo *MQReplyOptions `json:"replyTo,omitempty"`
	Auth    Credentials     `json:"credentials"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`

	// Whether to omit CloudEvent context attributes in messages sent to MQ.
	// When this property is false (default), the entire CloudEvent payload is included.
	// When this property is true, only the CloudEvent data is included.
	DiscardCEContext bool `json:"discardCloudEventContext"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

type MQReplyOptions struct {
	QueueManager string `json:"queueManager,omitempty"`
	QueueName    string `json:"queueName,omitempty"`
}

// Credentials holds the auth details.
type Credentials struct {
	User     v1alpha1.ValueFromField `json:"username,omitempty"`
	Password v1alpha1.ValueFromField `json:"password,omitempty"`
	TLS      *TLSSpec                `json:"tls,omitempty"`
}

// TLSSpec defines the desired state of the event target.
type TLSSpec struct {
	Cipher             string   `json:"cipher"`
	ClientAuthRequired bool     `json:"clientAuthRequired"`
	CertLabel          *string  `json:"certLabel,omitempty"`
	KeyRepository      Keystore `json:"keyRepository"`
}

// Keystore represents Key Database components.
type Keystore struct {
	KeyDatabase   v1alpha1.ValueFromField `json:"keyDatabase"`
	PasswordStash v1alpha1.ValueFromField `json:"passwordStash"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IBMMQTargetList is a list of event target instances.
type IBMMQTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []IBMMQTarget `json:"items"`
}
