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

// AWSEventBridgeSource is the Schema for the event source.
type AWSEventBridgeSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSEventBridgeSourceSpec   `json:"spec,omitempty"`
	Status AWSEventBridgeSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*AWSEventBridgeSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*AWSEventBridgeSource)(nil)
	_ v1alpha1.EventSource            = (*AWSEventBridgeSource)(nil)
	_ v1alpha1.EventSender            = (*AWSEventBridgeSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*AWSEventBridgeSource)(nil)
)

// AWSEventBridgeSourceSpec defines the desired state of the event source.
type AWSEventBridgeSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// EventBridge event bus ARN
	// https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazoneventbridge.html#amazoneventbridge-resources-for-iam-policies
	ARN apis.ARN `json:"arn"`

	// Event pattern used to select events that this source should subscribe to.
	// If not specified, the event rule is created with a catch-all pattern.
	// https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-event-patterns.html
	// +optional
	EventPattern *string `json:"eventPattern,omitempty"`

	// The intermediate destination of notifications originating from the
	// Amazon EventBridge event bus, before they are retrieved by this
	// event source.
	// If omitted, an Amazon SQS queue is automatically created and
	// associated with the EventBridge event rule.
	// +optional
	Destination *AWSEventBridgeSourceDestination `json:"destination,omitempty"`

	// Authentication method to interact with the Amazon S3 and SQS APIs.
	Auth v1alpha1.AWSAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// AWSEventBridgeSourceDestination contains possible intermediate destinations
// for the event bus' events.
type AWSEventBridgeSourceDestination struct {
	// Amazon SQS destination.
	// +optional
	SQS *AWSEventBridgeSourceDestinationSQS `json:"sqs,omitempty"`
}

// AWSEventBridgeSourceDestinationSQS contains properties of an Amazon SQS
// queue to use as destination for the event bus' events.
type AWSEventBridgeSourceDestinationSQS struct {
	// SQS Queue ARN
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/list_amazonsqs.html#amazonsqs-resources-for-iam-policies
	QueueARN apis.ARN `json:"queueARN"`
}

// AWSEventBridgeSourceStatus defines the observed state of the event source.
type AWSEventBridgeSourceStatus struct {
	v1alpha1.Status `json:",inline"`
	EventRuleARN    *apis.ARN `json:"ruleARN,omitempty"`
	QueueARN        *apis.ARN `json:"queueARN,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSEventBridgeSourceList contains a list of event sources.
type AWSEventBridgeSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSEventBridgeSource `json:"items"`
}
