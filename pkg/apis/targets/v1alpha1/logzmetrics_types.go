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

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LogzMetricsTarget receives CloudEvents typed `io.triggermesh.opentelemetry.metrics.push`
// that fullfil the schema at https://docs.triggermesh.io/schemas/opentelemetry.metrics.push.json
// to push new observations.
//
// The target works using an OpenTelemetry to Cortex adapter, and is able to manage
// OpenTelemetry Synchronous Kinds.
// In case of an error a CloudEvent response conformant with https://docs.triggermesh.io/schemas/triggermesh.error.json
// and with an the attribute extension `category: error` can be produced.
//
// Due to the buffering nature of this target, not returning an error does not guarantee that the
// metrics have been pushed to Logz
type LogzMetricsTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LogzMetricsTargetSpec   `json:"spec"`
	Status LogzMetricsTargetStatus `json:"status,omitempty"`
}

// Check the interfaces LogzMetricsTarget should be implementing.
var (
	_ runtime.Object            = (*LogzMetricsTarget)(nil)
	_ kmeta.OwnerRefable        = (*LogzMetricsTarget)(nil)
	_ targets.IntegrationTarget = (*LogzMetricsTarget)(nil)
	_ targets.EventSource       = (*LogzMetricsTarget)(nil)
	_ duckv1.KRShaped           = (*LogzMetricsTarget)(nil)
)

// LogzMetricsTargetSpec holds the desired state of the LogzMetricsTarget.
type LogzMetricsTargetSpec struct {

	// Connection information for LogzMetrics.
	Connection LogzMetricsConnection

	// Instruments configured for pushing metrics. It is mandatory that all metrics
	// pushed by using this target are pre-registered using this list.
	Instruments []Instrument

	// EventOptions for targets
	// +optional
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// LogzMetricsConnection contains the information to connect to a Logz tenant to push metrics.
type LogzMetricsConnection struct {
	// Token for connecting to Logz metrics listener.
	Token SecretValueFromSource `json:"token"`

	// ListenerURL for pushing metrics.
	ListenerURL string `json:"listenerURL"`
}

// InstrumentKind as defined by OpenTelemetry.
type InstrumentKind string

// NumberKind as defined by OpenTelemetry.
type NumberKind string

const (
	// Instrument Kinds
	InstrumentKindHistogram     InstrumentKind = "Histogram"
	InstrumentKindCounter       InstrumentKind = "Counter"
	InstrumentKindUpDownCounter InstrumentKind = "UpDownCounter"

	// Number Kinds
	NumberKindInt64   NumberKind = "Int64"
	NumberKindFloat64 NumberKind = "Float64"
)

// Instrument push metrics for.
type Instrument struct {
	// Name for the Instrument.
	Name string

	// Description for the Instrument
	// +optional
	Description *string

	// Instrument Kind as defined by OpenTelemetry. Supported values are:
	//
	// - Histogram: for absolute values that can be aggregated.
	// - Counter: for delta values that increase monotonically.
	// - UpDownCounter: for delta values that can increase and decrease.
	Instrument InstrumentKind

	// Number Kind as defined by OpenTelemetry, defines the measure data type
	// accepted by the Instrument. Supported values are:
	//
	// - Int64.
	// - Float64.
	Number NumberKind
}

// LogzMetricsTargetStatus communicates the observed state of the LogzMetricsTarget from the controller.
type LogzMetricsTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`
	CloudEventStatus     `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LogzMetricsTargetList is a list of LogzMetricsTarget resources.
type LogzMetricsTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []LogzMetricsTarget `json:"items"`
}
