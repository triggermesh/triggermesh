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

// AWSS3Source is the Schema for the event source.
type AWSS3Source struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSS3SourceSpec   `json:"spec,omitempty"`
	Status AWSS3SourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*AWSS3Source)(nil)
	_ v1alpha1.AdapterConfigurable    = (*AWSS3Source)(nil)
	_ v1alpha1.EventSource            = (*AWSS3Source)(nil)
	_ v1alpha1.EventSender            = (*AWSS3Source)(nil)
	_ v1alpha1.ServiceAccountProvider = (*AWSS3Source)(nil)
)

// AWSS3SourceSpec defines the desired state of the event source.
type AWSS3SourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Bucket ARN
	// https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazons3.html#amazons3-resources-for-iam-policies
	//
	// Although not technically supported by S3, the ARN provided via this
	// attribute may include a region and an account ID. When this
	// information is provided, it is used to set an accurate
	// identity-based access policy between the S3 bucket and the
	// reconciled SQS queue, unless an existing queue is provided via the
	// QueueARN attribute.
	ARN apis.ARN `json:"arn"`

	// List of event types that the source should subscribe to.
	// Accepted values:
	// https://docs.aws.amazon.com/AmazonS3/latest/API/API_QueueConfiguration.html
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-how-to-event-types-and-destinations.html
	EventTypes []string `json:"eventTypes"`

	// The intermediate destination of notifications originating from the
	// Amazon S3 bucket, before they are retrieved by this event source.
	// If omitted, an Amazon SQS queue is automatically created and
	// associated with the bucket.
	// +optional
	Destination *AWSS3SourceDestination `json:"destination,omitempty"`

	// Authentication method to interact with the Amazon S3 and SQS APIs.
	Auth v1alpha1.AWSAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// AWSS3SourceDestination contains possible intermediate destinations for
// bucket notifications.
type AWSS3SourceDestination struct {
	// Amazon SQS destination.
	// +optional
	SQS *AWSS3SourceDestinationSQS `json:"sqs,omitempty"`
}

// AWSS3SourceDestinationSQS contains properties of an Amazon SQS queue to use
// as destination for bucket notifications.
type AWSS3SourceDestinationSQS struct {
	// SQS Queue ARN
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/list_amazonsqs.html#amazonsqs-resources-for-iam-policies
	QueueARN apis.ARN `json:"queueARN"`
}

// AWSS3SourceStatus defines the observed state of the event source.
type AWSS3SourceStatus struct {
	v1alpha1.Status `json:",inline"`
	QueueARN        *apis.ARN `json:"queueARN,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSS3SourceList contains a list of event sources.
type AWSS3SourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSS3Source `json:"items"`
}
