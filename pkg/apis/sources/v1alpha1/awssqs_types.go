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

// AWSSQSSource is the Schema for the event source.
type AWSSQSSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSSQSSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status  `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*AWSSQSSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*AWSSQSSource)(nil)
	_ v1alpha1.EventSource            = (*AWSSQSSource)(nil)
	_ v1alpha1.EventSender            = (*AWSSQSSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*AWSSQSSource)(nil)
)

// AWSSQSSourceSpec defines the desired state of the event source.
type AWSSQSSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Queue ARN
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/list_amazonsqs.html#amazonsqs-resources-for-iam-policies
	ARN apis.ARN `json:"arn"`

	// Options that control the behavior of message receivers.
	// +optional
	ReceiveOptions *AWSSQSSourceReceiveOptions `json:"receiveOptions,omitempty"`

	// Name of the message processor to use for converting SQS messages to CloudEvents.
	// Supported values are "default" and "s3".
	// +optional
	MessageProcessor *string `json:"messageProcessor,omitempty"`

	// Authentication method to interact with the Amazon SQS API.
	Auth v1alpha1.AWSAuth `json:"auth"`

	// Customizations of the AWS REST API endpoint.
	// +optional
	Endpoint *v1alpha1.AWSEndpoint `json:"endpoint,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// AWSSQSSourceReceiveOptions defines options that control the behavior of
// Amazon SQS message receivers.
type AWSSQSSourceReceiveOptions struct {
	// Period of time during which Amazon SQS prevents other consumers from
	// receiving and processing a message that has been received via ReceiveMessage.
	// Expressed as a duration string, which format is documented at https://pkg.go.dev/time#ParseDuration.
	//
	// If not defined, the overall visibility timeout for the queue is used.
	//
	// For more details, please refer to the Amazon SQS Developer Guide at
	// https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html.
	//
	// +optional
	VisibilityTimeout *apis.Duration `json:"visibilityTimeout,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSSQSSourceList contains a list of event sources.
type AWSSQSSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSSQSSource `json:"items"`
}
