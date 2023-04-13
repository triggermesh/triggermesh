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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *AWSSNSSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AWSSNSSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (s *AWSSNSSource) GetConditionSet() apis.ConditionSet {
	return awsSNSSourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *AWSSNSSource) GetStatus() *duckv1.Status {
	return &s.Status.Status.Status
}

// GetSink implements EventSender.
func (s *AWSSNSSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *AWSSNSSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status.Status,
	}
}

// Supported event types
const (
	AWSSNSGenericEventType = "notification"
)

// GetEventTypes implements EventSource.
func (s *AWSSNSSource) GetEventTypes() []string {
	return []string{
		AWSEventType(s.Spec.ARN.Service, AWSSNSGenericEventType),
	}
}

// AsEventSource implements EventSource.
func (s *AWSSNSSource) AsEventSource() string {
	return s.Spec.ARN.String()
}

// IsMultiTenant implements MultiTenant.
func (*AWSSNSSource) IsMultiTenant() bool {
	return true
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *AWSSNSSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// Status conditions
const (
	// AWSSNSConditionSubscribed has status True when the event source's HTTP(S) endpoint has been subscribed to the
	// SNS subscription.
	AWSSNSConditionSubscribed apis.ConditionType = "Subscribed"

	// AWSSNSConditionHandlerRegistered indicates that a HTTP handler was registered for the source object.
	// It is not part of the ConditionSet registered for the AWSSNSSource
	// type, and will therefore automatically be propagated by Knative with
	// a severity of "Info".
	AWSSNSConditionHandlerRegistered = "HandlerRegistered"
)

// Reasons for status conditions
const (
	// AWSSNSReasonNoURL is set on a Subscribed condition when the adapter URL is empty.
	AWSSNSReasonNoURL = "MissingAdapterURL"
	// AWSSNSReasonNoClient is set on a Subscribed condition when a SNS API client cannot be obtained.
	AWSSNSReasonNoClient = "NoClient"
	// AWSSNSReasonPending is set on a Subscribed condition when the SNS subscription is pending confirmation.
	AWSSNSReasonPending = "PendingConfirmation"
	// AWSSNSReasonRejected is set on a Subscribed condition when the SNS API rejects a subscription request.
	AWSSNSReasonRejected = "SubscriptionRejected"
	// AWSSNSReasonAPIError is set on a Subscribed condition when the SNS API returns any other error.
	AWSSNSReasonAPIError = "APIError"
	// AWSSNSReasonFailedSync is set on a Subscribed condition when other synchronization errors occur.
	AWSSNSReasonFailedSync = "FailedSync"
)

// awsSNSSourceConditionSet is a set of conditions for AWSSNSSource objects.
var awsSNSSourceConditionSet = v1alpha1.NewConditionSet(
	AWSSNSConditionSubscribed,
)

// MarkSubscribed sets the Subscribed condition to True and reports the ARN of
// the SNS subscription.
func (s *AWSSNSSourceStatus) MarkSubscribed(subARN string) {
	s.SubscriptionARN = &subARN
	awsSNSSourceConditionSet.Manage(s).MarkTrue(AWSSNSConditionSubscribed)
}

// MarkNotSubscribed sets the Subscribed condition to False with the given
// reason and associated message.
func (s *AWSSNSSourceStatus) MarkNotSubscribed(reason, msg string) {
	awsSNSSourceConditionSet.Manage(s).MarkFalse(AWSSNSConditionSubscribed,
		reason, msg)
}

// SetDefaults implements apis.Defaultable
func (s *AWSSNSSource) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (s *AWSSNSSource) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
