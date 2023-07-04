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
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	tmapis "github.com/triggermesh/triggermesh/pkg/apis"
	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *AWSEventBridgeSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AWSEventBridgeSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (s *AWSEventBridgeSource) GetConditionSet() apis.ConditionSet {
	return awsEventBridgeSourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *AWSEventBridgeSource) GetStatus() *duckv1.Status {
	return &s.Status.Status.Status
}

// GetSink implements EventSender.
func (s *AWSEventBridgeSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *AWSEventBridgeSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status.Status,
	}
}

// Supported event types
const (
	AWSEventBridgeGenericEventType = "event"
)

// GetEventTypes implements EventSource.
func (s *AWSEventBridgeSource) GetEventTypes() []string {
	return []string{
		AWSEventType(s.Spec.ARN.Service, AWSEventBridgeGenericEventType),
	}
}

// AsEventSource implements EventSource.
func (s *AWSEventBridgeSource) AsEventSource() string {
	return s.Spec.ARN.String()
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *AWSEventBridgeSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// WantsOwnServiceAccount implements ServiceAccountProvider.
func (s *AWSEventBridgeSource) WantsOwnServiceAccount() bool {
	return s.Spec.Auth.WantsOwnServiceAccount()
}

// ServiceAccountOptions implements ServiceAccountProvider.
func (s *AWSEventBridgeSource) ServiceAccountOptions() []resource.ServiceAccountOption {
	return s.Spec.Auth.ServiceAccountOptions()
}

// Status conditions
const (
	// AWSEventBridgeConditionSubscribed has status True when events from an EventBridge
	// event bus have been successfully subscribed to via a rule.
	AWSEventBridgeConditionSubscribed apis.ConditionType = "Subscribed"
)

// Reasons for status conditions
const (
	// AWSEventBridgeReasonNoClient is set on a Subscribed condition when an EventBridge/SQS API client cannot be obtained.
	AWSEventBridgeReasonNoClient = "NoClient"
	// AWSEventBridgeReasonNoEventBus is set on a Subscribed condition when the EventBridge event bus does not exist.
	AWSEventBridgeReasonNoEventBus = "EventBusNotFound"
	// AWSEventBridgeReasonInvalidEventPattern is set on a Subscribed condition when the provided event pattern is invalid.
	AWSEventBridgeReasonInvalidEventPattern = "InvalidEventPattern"
	// AWSEventBridgeReasonAPIError is set on a Subscribed condition when the EventBridge/SQS API returns any other error.
	AWSEventBridgeReasonAPIError = "APIError"
)

// awsEventBridgeSourceConditionSet is a set of conditions for AWSEventBridgeSource objects.
var awsEventBridgeSourceConditionSet = v1alpha1.NewConditionSet(
	AWSEventBridgeConditionSubscribed,
)

// MarkSubscribed sets the Subscribed condition to True and reports the ARN of
// the EventBridge event rule.
func (s *AWSEventBridgeSourceStatus) MarkSubscribed(ruleARN tmapis.ARN) {
	s.EventRuleARN = &ruleARN
	awsEventBridgeSourceConditionSet.Manage(s).MarkTrue(AWSEventBridgeConditionSubscribed)
}

// MarkNotSubscribed sets the Subscribed condition to False with the given
// reason and associated message.
func (s *AWSEventBridgeSourceStatus) MarkNotSubscribed(reason, msg string) {
	awsEventBridgeSourceConditionSet.Manage(s).MarkFalse(AWSEventBridgeConditionSubscribed,
		reason, msg)
}

// SetDefaults implements apis.Defaultable
func (s *AWSEventBridgeSource) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (s *AWSEventBridgeSource) Validate(ctx context.Context) *apis.FieldError {
	// Do not validate authentication object in case of resource deletion
	if s.DeletionTimestamp != nil {
		return nil
	}
	return s.Spec.Auth.Validate(ctx)
}
