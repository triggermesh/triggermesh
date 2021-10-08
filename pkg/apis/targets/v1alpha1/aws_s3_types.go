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

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSS3Target is the Schema for an AWS s3 Target.
type AWSS3Target struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the AWSS3Target (from the client).
	Spec AWSS3TargetSpec `json:"spec"`

	// Status communicates the observed state of the AWSS3Target (from the controller).
	Status AWSS3TargetStatus `json:"status,omitempty"`
}

// Check the interfaces AWSS3Target should be implementing.
var (
	_ runtime.Object            = (*AWSS3Target)(nil)
	_ kmeta.OwnerRefable        = (*AWSS3Target)(nil)
	_ targets.IntegrationTarget = (*AWSS3Target)(nil)
	_ targets.EventSource       = (*AWSS3Target)(nil)
	_ duckv1.KRShaped           = (*AWSS3Target)(nil)
)

// AWSS3TargetSpec holds the desired state of the even target.
type AWSS3TargetSpec struct {
	// AWS account Key
	AWSApiKey SecretValueFromSource `json:"awsApiKey"`

	// AWS account secret key
	AWSApiSecret SecretValueFromSource `json:"awsApiSecret"`

	// Amazon Resource Name of the S3 bucket.
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/list_amazons3.html#amazons3-resources-for-iam-policies
	ARN string `json:"arn"`

	// Whether to omit CloudEvent context attributes in created S3 objects.
	// When this property is false (default), the entire CloudEvent payload is included.
	// When this property is true, only the CloudEvent data is included.
	DiscardCEContext bool `json:"discardCloudEventContext"`
}

// AWSS3TargetStatus communicates the observed state of the event target.
type AWSS3TargetStatus struct {
	AWSTargetStatus `json:",inline"`
	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSS3TargetList is a list of AWSS3Target resources
type AWSS3TargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AWSS3Target `json:"items"`
}
