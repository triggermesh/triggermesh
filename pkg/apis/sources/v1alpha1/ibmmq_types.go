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

// IBMMQSource is the Schema the event source.
type IBMMQSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IBMMQSourceSpec `json:"spec"`
	Status v1alpha1.Status `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*IBMMQSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*IBMMQSource)(nil)
	_ v1alpha1.EventSource         = (*IBMMQSource)(nil)
	_ v1alpha1.EventSender         = (*IBMMQSource)(nil)
)

// IBMMQSourceSpec defines the desired state of the event source.
type IBMMQSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	ConnectionName string `json:"connectionName"`
	QueueManager   string `json:"queueManager"`
	QueueName      string `json:"queueName"`
	ChannelName    string `json:"channelName"`

	Delivery Delivery `json:"delivery,omitempty"`

	Auth Credentials `json:"credentials"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// Delivery defines the source's message delivery behavior.
type Delivery struct {
	DeadLetterQueue string `json:"deadLetterQueue"`
	Retry           int    `json:"retry"`

	// currently not used
	DeadLetterQueueManager string `json:"deadLetterQueueManager,omitempty"`
	BackoffDelay           int    `json:"backoffDelay,omitempty"`
}

// Credentials holds the auth details.
type Credentials struct {
	User     v1alpha1.ValueFromField `json:"username,omitempty"`
	Password v1alpha1.ValueFromField `json:"password,omitempty"`
	TLS      *TLSSpec                `json:"tls,omitempty"`
}

// TLSSpec holds the IBM MQ TLS authentication parameters.
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

// IBMMQSourceList is a list of event source instances.
type IBMMQSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []IBMMQSource `json:"items"`
}
