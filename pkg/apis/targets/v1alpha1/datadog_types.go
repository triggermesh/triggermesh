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

// DatadogTarget is the Schema for an HTTP Target.
type DatadogTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatadogTargetSpec `json:"spec"`
	Status v1alpha1.Status   `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*DatadogTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*DatadogTarget)(nil)
	_ v1alpha1.EventReceiver       = (*DatadogTarget)(nil)
	_ v1alpha1.EventSource         = (*DatadogTarget)(nil)
)

// DatadogTargetSpec defines the desired state of the event target.
type DatadogTargetSpec struct {
	// DatadogApiKey represents how Datadog credentials should be provided in the secret
	DatadogAPIKey SecretValueFromSource `json:"apiKey"`

	// DatadogSite controls the site of the Datadog intake API, defaults to `datadoghq.com`
	// +optional
	DatadogSite *string `json:"site,omitempty"`

	// MetricPrefix is prepended to the name of the associated metrics.
	// +optional
	MetricPrefix *string `json:"metricPrefix"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatadogTargetList is a list of event target instances.
type DatadogTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []DatadogTarget `json:"items"`
}
