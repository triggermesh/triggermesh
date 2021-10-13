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

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSKinesisTarget is the Schema for an AWS Kinesis Target.
type AWSKinesisTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the AWSKinesisTarget (from the client).
	Spec AWSKinesisTargetSpec `json:"spec"`

	// Status communicates the observed state of the AWSKinesisTarget (from the controller).
	Status AWSTargetStatus `json:"status,omitempty"`
}

// Check the interfaces AWSKinesisTarget should be implementing.
var (
	_ runtime.Object     = (*AWSKinesisTarget)(nil)
	_ kmeta.OwnerRefable = (*AWSKinesisTarget)(nil)
	_ duckv1.KRShaped    = (*AWSKinesisTarget)(nil)
)

// AWSKinesisTargetSpec holds the desired state of the event target.
type AWSKinesisTargetSpec struct {
	// AWS account Key
	AWSApiKey SecretValueFromSource `json:"awsApiKey"`

	// AWS account secret key
	AWSApiSecret SecretValueFromSource `json:"awsApiSecret"`

	// Amazon Resource Name of the Kinesis stream.
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/list_amazonkinesis.html#amazonkinesis-resources-for-iam-policies
	ARN string `json:"arn"`

	// Kinesis Partition to publish the events to
	Partition string `json:"partition"`

	// Whether to omit CloudEvent context attributes in created Kinesis records.
	// When this property is false (default), the entire CloudEvent payload is included.
	// When this property is true, only the CloudEvent data is included.
	DiscardCEContext bool `json:"discardCloudEventContext"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSKinesisTargetList is a list of AWSKinesisTarget resources
type AWSKinesisTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AWSKinesisTarget `json:"items"`
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *AWSKinesisTarget) GetConditionSet() apis.ConditionSet {
	return AwsCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *AWSKinesisTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
