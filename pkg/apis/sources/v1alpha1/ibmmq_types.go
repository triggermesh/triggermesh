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
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IBMMQSource is the Schema the event source.
type IBMMQSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IBMMQSourceSpec   `json:"spec"`
	Status IBMMQSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ runtime.Object = (*IBMMQSource)(nil)
	_ EventSource    = (*IBMMQSource)(nil)
)

// IBMMQSourceSpec holds the desired state of the event source.
type IBMMQSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	ConnectionName string `json:"connectionName"`
	QueueManager   string `json:"queueManager"`
	QueueName      string `json:"queueName"`
	ChannelName    string `json:"channelName"`

	Delivery Delivery `json:"delivery,omitempty"`

	Auth Credentials `json:"credentials"`
}

// Delivery defines the source's message delivery behavior
type Delivery struct {
	DeadLetterQueueManager string `json:"deadLetterQueueManager"`
	DeadLetterQueue        string `json:"deadLetterQueue"`
	Retry                  int    `json:"retry"`
	// currently not used
	BackoffDelay int `json:"backoffDelay"`
}

// Credentials holds the auth details
type Credentials struct {
	User     ValueFromField `json:"username"`
	Password ValueFromField `json:"password"`
}

// IBMMQSourceStatus defines the observed state of the event source.
type IBMMQSourceStatus struct {
	EventSourceStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IBMMQSourceList is a list of event source instances.
type IBMMQSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []IBMMQSource `json:"items"`
}
