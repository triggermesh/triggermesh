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

// AWSSQSTarget is the Schema for an AWS SQS Target.
type AWSSQSTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSSQSTargetSpec `json:"spec"`
	Status v1alpha1.Status  `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*AWSSQSTarget)(nil)
	_ v1alpha1.AdapterConfigurable    = (*AWSSQSTarget)(nil)
	_ v1alpha1.ServiceAccountProvider = (*AWSSQSTarget)(nil)
)

// AWSSQSTargetSpec defines the desired state of the event target.
type AWSSQSTargetSpec struct {
	// AWS-specific authentication methods.
	Auth v1alpha1.AWSAuth `json:"auth"`

	// Amazon Resource Name of the SQS queue.
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/list_amazonsqs.html#amazonsqs-resources-for-iam-policies
	ARN string `json:"arn"`

	// Message Group ID is required for FIFO based queues, and is used to uniquely identify the event producer
	// https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/FIFO-queues-understanding-logic.html
	// +optional
	MessageGroupID string `json:"messageGroupId,omitempty"`

	// Whether to omit CloudEvent context attributes in messages sent to SQS.
	// When this property is false (default), the entire CloudEvent payload is included.
	// When this property is true, only the CloudEvent data is included.
	DiscardCEContext bool `json:"discardCloudEventContext"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSSQSTargetList is a list of event target instances.
type AWSSQSTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AWSSQSTarget `json:"items"`
}
