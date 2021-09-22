/*
Copyright (c) 2021 TriggerMesh Inc.

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

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SplunkTarget is the Schema for the event target.
type SplunkTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SplunkTargetSpec   `json:"spec,omitempty"`
	Status SplunkTargetStatus `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ runtime.Object     = (*SplunkTarget)(nil)
	_ kmeta.OwnerRefable = (*SplunkTarget)(nil)
	_ duckv1.KRShaped    = (*SplunkTarget)(nil)
)

// SplunkTargetSpec defines the desired state of the event target.
type SplunkTargetSpec struct {
	// URL of the HTTP Event Collector (HEC).
	// Only the scheme, hostname, and port (optionally) are evaluated, the URL path is trimmed if present.
	// see https://docs.splunk.com/Documentation/Splunk/latest/Data/UsetheHTTPEventCollector#Enable_HTTP_Event_Collector
	Endpoint apis.URL `json:"endpoint"`
	// Token for authenticating requests against the HEC.
	// see https://docs.splunk.com/Documentation/Splunk/latest/Data/UsetheHTTPEventCollector#About_Event_Collector_tokens
	Token ValueFromField `json:"token"`
	// Name of the index to send events to.
	// When undefined, events are sent to the default index defined in the HEC token's configuration.
	// +optional
	Index *string `json:"index,omitempty"`

	// Controls whether the Splunk client verifies the server's certificate
	// chain and host name when communicating over TLS.
	// +optional
	SkipTLSVerify *bool `json:"skipTLSVerify,omitempty"`
}

// SplunkTargetStatus defines the observed state of the event target.
type SplunkTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SplunkTargetList contains a list of event targets.
type SplunkTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SplunkTarget `json:"items"`
}
