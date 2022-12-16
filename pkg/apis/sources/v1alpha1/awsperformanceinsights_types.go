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

// AWSPerformanceInsightsSource is the Schema for the event source.
type AWSPerformanceInsightsSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSPerformanceInsightsSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status                  `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*AWSPerformanceInsightsSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*AWSPerformanceInsightsSource)(nil)
	_ v1alpha1.EventSource            = (*AWSPerformanceInsightsSource)(nil)
	_ v1alpha1.EventSender            = (*AWSPerformanceInsightsSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*AWSPerformanceInsightsSource)(nil)
)

// AWSPerformanceInsightsSourceSpec defines the desired state of the event source.
type AWSPerformanceInsightsSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// ARN of the RDS instance to receive metrics for.
	// https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazonrds.html#amazonrds-resources-for-iam-policies
	ARN apis.ARN `json:"arn"`

	// Duration which defines how often metrics should be pulled from Amazon Performance Insights.
	// Expressed as a duration string, which format is documented at https://pkg.go.dev/time#ParseDuration.
	PollingInterval apis.Duration `json:"pollingInterval"`

	// List of queries that determine what metrics will be sourced from Amazon Performance Insights.
	//
	// Each item represents the 'metric' attribute of a MetricQuery.
	// https://docs.aws.amazon.com/performance-insights/latest/APIReference/API_MetricQuery.html
	Metrics []string `json:"metrics"`

	// Authentication method to interact with the Amazon RDS and Performance Insights APIs.
	Auth v1alpha1.AWSAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSPerformanceInsightsSourceList contains a list of event sources.
type AWSPerformanceInsightsSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSPerformanceInsightsSource `json:"items"`
}
