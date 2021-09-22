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
	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatadogTarget is the Schema for an HTTP Target.
type DatadogTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatadogTargetSpec   `json:"spec"`
	Status DatadogTargetStatus `json:"status,omitempty"`
}

// Check the interfaces DatadogTarget should be implementing.
var (
	_ runtime.Object            = (*DatadogTarget)(nil)
	_ kmeta.OwnerRefable        = (*DatadogTarget)(nil)
	_ targets.IntegrationTarget = (*DatadogTarget)(nil)
	_ targets.EventSource       = (*DatadogTarget)(nil)
	_ duckv1.KRShaped           = (*DatadogTarget)(nil)
)

// DatadogTargetSpec holds the desired state of the DatadogTarget.
type DatadogTargetSpec struct {
	// DatadogApiKey represents how Datadog credentials should be provided in the secret
	DatadogAPIKey SecretValueFromSource `json:"apiKey"`

	// MetricPrefix is prepended to the name of the associated metrics.
	// +optional
	MetricPrefix *string `json:"metricPrefix"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// DatadogTargetStatus communicates the observed state of the DatadogTarget (from the controller).
type DatadogTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`

	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatadogTargetList is a list of DatadogTarget resources
type DatadogTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []DatadogTarget `json:"items"`
}
