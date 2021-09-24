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
	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*GoogleCloudBillingSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GoogleCloudBillingSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (*GoogleCloudBillingSource) GetConditionSet() apis.ConditionSet {
	return GoogleCloudBillingSourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *GoogleCloudBillingSource) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetSink implements EventSource.
func (s *GoogleCloudBillingSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements EventSource.
func (s *GoogleCloudBillingSource) GetStatusManager() *EventSourceStatusManager {
	return &EventSourceStatusManager{
		ConditionSet:      s.GetConditionSet(),
		EventSourceStatus: &s.Status.EventSourceStatus,
	}
}

// AsEventSource implements EventSource.
func (s *GoogleCloudBillingSource) AsEventSource() string {
	return s.Spec.BudgetId
}

// Supported event types
const (
	GoogleCloudBillingGenericEventType = "com.google.cloud.billing.notification"
)

// GetEventTypes returns the event types generated by the source.
func (*GoogleCloudBillingSource) GetEventTypes() []string {
	return []string{
		GoogleCloudBillingGenericEventType,
	}
}

// Status conditions
const (
	// GoogleCloudBillingConditionSubscribed has status True when the source has subscribed to a topic.
	GoogleCloudBillingConditionSubscribed apis.ConditionType = "Subscribed"
)

// GoogleCloudBillingSourceConditionSet is a set of conditions for
// GoogleCloudBillingSource objects.
var GoogleCloudBillingSourceConditionSet = NewEventSourceConditionSet(
	GoogleCloudBillingConditionSubscribed,
)

// MarkSubscribed sets the Subscribed condition to True.
func (s *GoogleCloudBillingSourceStatus) MarkSubscribed() {
	GoogleCloudBillingSourceConditionSet.Manage(s).MarkTrue(GoogleCloudBillingConditionSubscribed)
}

// MarkNotSubscribed sets the Subscribed condition to False with the given
// reason and message.
func (s *GoogleCloudBillingSourceStatus) MarkNotSubscribed(reason, msg string) {
	s.Topic = nil
	s.Subscription = nil
	GoogleCloudBillingSourceConditionSet.Manage(s).MarkFalse(GoogleCloudBillingConditionSubscribed, reason, msg)
}
