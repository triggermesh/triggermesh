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

	"github.com/triggermesh/triggermesh/pkg/apis"
	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSCloudWatchSource is the Schema for the event source.
type AWSCloudWatchSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSCloudWatchSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status         `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*AWSCloudWatchSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*AWSCloudWatchSource)(nil)
	_ v1alpha1.EventSource            = (*AWSCloudWatchSource)(nil)
	_ v1alpha1.EventSender            = (*AWSCloudWatchSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*AWSCloudWatchSource)(nil)
)

// AWSCloudWatchSourceSpec defines the desired state of the event source.
type AWSCloudWatchSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Code of the AWS region to source metrics from.
	// Available region codes are documented at
	// https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints.
	Region string `json:"region"`

	// Duration which defines how often metrics should be pulled from Amazon CloudWatch.
	// Expressed as a duration string, which format is documented at https://pkg.go.dev/time#ParseDuration.
	//
	// Defaults to 5m
	//
	// +optional
	PollingInterval *apis.Duration `json:"pollingInterval,omitempty"`

	// List of queries that determine what metrics will be sourced from Amazon CloudWatch.
	// +optional
	MetricQueries []AWSCloudWatchMetricQuery `json:"metricQueries,omitempty"`

	// Authentication method to interact with the Amazon CloudWatch API.
	Auth v1alpha1.AWSAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// AWSCloudWatchMetricQuery represents a CloudWatch MetricDataQuery.
// https://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_MetricDataQuery.html
type AWSCloudWatchMetricQuery struct {
	// Unique short name that identifies the query.
	Name string `json:"name"`

	// Optional: no more than one of the following may be specified.

	// Math expression to be performed on the metric data.
	// +optional
	Expression *string `json:"expression,omitempty"`
	// Representation of a metric with statistics, period, and units, but no math expression.
	// +optional
	Metric *AWSCloudWatchMetricStat `json:"metric,omitempty"`
}

// AWSCloudWatchMetricStat is a representation of a metric with statistics,
// period, and units, but no math expression.
type AWSCloudWatchMetricStat struct {
	Metric AWSCloudWatchMetric `json:"metric"`         // Definition of the metric
	Period int64               `json:"period"`         // metric resolution in seconds
	Stat   string              `json:"stat"`           // statistic type to use
	Unit   string              `json:"unit,omitempty"` // The unit of the metric being returned
}

// AWSCloudWatchMetric is a metric definition.
type AWSCloudWatchMetric struct {
	Dimensions []AWSCloudWatchMetricDimension `json:"dimensions"`
	MetricName string                         `json:"metricName"`
	Namespace  string                         `json:"namespace"`
}

// AWSCloudWatchMetricDimension represents the dimensions of a metric.
type AWSCloudWatchMetricDimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSCloudWatchSourceList contains a list of event sources.
type AWSCloudWatchSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSCloudWatchSource `json:"items"`
}
