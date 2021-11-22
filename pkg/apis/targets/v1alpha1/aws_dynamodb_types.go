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

// AWSDynamoDBTarget is the Schema for an AWS DynamoDB Target.
type AWSDynamoDBTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the AWSDynamoDBTarget (from the client).
	Spec AWSDynamoDBTargetSpec `json:"spec"`

	// Status communicates the observed state of the AWSDynamoDBTarget (from the controller).
	Status AWSDynamoDBTargetStatus `json:"status,omitempty"`
}

// Check the interfaces AWSDynamoDBTarget should be implementing.
var (
	_ runtime.Object      = (*AWSDynamoDBTarget)(nil)
	_ kmeta.OwnerRefable  = (*AWSDynamoDBTarget)(nil)
	_ targets.EventSource = (*AWSDynamoDBTarget)(nil)
	_ duckv1.KRShaped     = (*AWSDynamoDBTarget)(nil)
)

// AWSDynamoDBTargetSpec holds the desired state of the event target.
type AWSDynamoDBTargetSpec struct {
	// AWS account Key
	AWSApiKey SecretValueFromSource `json:"awsApiKey"`

	// AWS account secret key
	AWSApiSecret SecretValueFromSource `json:"awsApiSecret"`

	// Table ARN
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/list_amazondynamodb.html#amazondynamodb-resources-for-iam-policies
	ARN string `json:"arn"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// AWSDynamoDBTargetStatus communicates the observed state of the event target.
type AWSDynamoDBTargetStatus struct {
	AWSTargetStatus `json:",inline"`
	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSDynamoDBTargetList is a list of AWSDynamoDBTarget resources
type AWSDynamoDBTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AWSDynamoDBTarget `json:"items"`
}
