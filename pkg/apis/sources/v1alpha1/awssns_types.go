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

// AWSSNSSource is the Schema for the event source.
type AWSSNSSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSSNSSourceSpec   `json:"spec,omitempty"`
	Status AWSSNSSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AWSSNSSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*AWSSNSSource)(nil)
	_ v1alpha1.MultiTenant         = (*AWSSNSSource)(nil)
	_ v1alpha1.EventSource         = (*AWSSNSSource)(nil)
	_ v1alpha1.EventSender         = (*AWSSNSSource)(nil)
)

// AWSSNSSourceSpec defines the desired state of the event source.
type AWSSNSSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Topic ARN
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/list_amazonsns.html#amazonsns-resources-for-iam-policies
	ARN apis.ARN `json:"arn"`

	// Attributes to set on the Subscription that is used for receiving messages from the topic.
	// For a list of supported subscription attributes, please refer to the following resources:
	//  * https://docs.aws.amazon.com/sns/latest/api/API_SetSubscriptionAttributes.html
	//  * https://docs.aws.amazon.com/sns/latest/dg/sns-how-it-works.html
	// +optional
	SubscriptionAttributes map[string]*string `json:"subscriptionAttributes,omitempty"`

	// Authentication method to interact with the Amazon SNS API.
	Auth v1alpha1.AWSAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// AWSSNSSourceStatus defines the observed state of the event source.
type AWSSNSSourceStatus struct {
	v1alpha1.Status `json:",inline"`
	SubscriptionARN *string `json:"subscriptionARN,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSSNSSourceList contains a list of event sources.
type AWSSNSSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSSNSSource `json:"items"`
}
