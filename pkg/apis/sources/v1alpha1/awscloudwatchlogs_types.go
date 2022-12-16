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

// AWSCloudWatchLogsSource is the Schema for the event source.
type AWSCloudWatchLogsSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSCloudWatchLogsSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status             `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*AWSCloudWatchLogsSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*AWSCloudWatchLogsSource)(nil)
	_ v1alpha1.EventSource            = (*AWSCloudWatchLogsSource)(nil)
	_ v1alpha1.EventSender            = (*AWSCloudWatchLogsSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*AWSCloudWatchLogsSource)(nil)
)

// AWSCloudWatchLogsSourceSpec defines the desired state of the event source.
type AWSCloudWatchLogsSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// ARN of the Log Group to source data from.
	// https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazoncloudwatchlogs.html#amazoncloudwatchlogs-resources-for-iam-policies
	ARN apis.ARN `json:"arn"`

	// Duration which defines how often logs should be pulled from Amazon CloudWatch Logs.
	// Expressed as a duration string, which format is documented at https://pkg.go.dev/time#ParseDuration.
	//
	// Defaults to 5m
	//
	// +optional
	PollingInterval *apis.Duration `json:"pollingInterval,omitempty"`

	// Authentication method to interact with the Amazon CloudWatch Logs API.
	Auth v1alpha1.AWSAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSCloudWatchLogsSourceList contains a list of event sources.
type AWSCloudWatchLogsSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSCloudWatchLogsSource `json:"items"`
}
