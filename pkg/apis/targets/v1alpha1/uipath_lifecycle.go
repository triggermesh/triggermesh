/*
Copyright (c) 2021 TriggerMesh Inc.

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
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// Accepted event types
const (
	// EventTypeUiPathStartJob represents job data to be initiated
	EventTypeUiPathStartJob = "io.triggermesh.uipath.job.start"
	// EventTypeUiPathQueuePost represents queue data to be posted to UiPath
	EventTypeUiPathQueuePost = "io.triggermesh.uipath.queue.post"
)

// UiPathCondSet is the group of possible conditions
var UiPathCondSet = apis.NewLivingConditionSet(
	ConditionServiceReady,
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *UiPathTargetStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return UiPathCondSet.Manage(s).GetCondition(t)
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *UiPathTargetStatus) InitializeConditions() {
	UiPathCondSet.Manage(s).InitializeConditions()
	s.Address = &duckv1.Addressable{}
}

// GetGroupVersionKind returns the GroupVersionKind.
func (*UiPathTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("UiPathTarget")
}

// AsEventSource implements targets.EventSource.
func (s *UiPathTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

// AcceptedEventTypes implements IntegrationTarget.
func (*UiPathTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeUiPathStartJob,
		EventTypeUiPathQueuePost,
	}
}

// GetEventTypes implements EventSource.
func (*UiPathTarget) GetEventTypes() []string {
	return []string{
		EventTypeResponse,
	}
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionServiceReady should be marked as true or false.
func (s *UiPathTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc != nil && ksvc.IsReady() {
		s.Address = ksvc.Status.Address
		UiPathCondSet.Manage(s).MarkTrue(ConditionServiceReady)
		return
	} else if ksvc == nil {
		s.MarkNoKService(ReasonUnavailable, "Adapter service unknown: ksvc is not available")
	} else {
		s.MarkNoKService(ReasonUnavailable, "Adapter service \"%s/%s\" is unavailable", ksvc.Namespace, ksvc.Name)
	}
	s.Address.URL = nil
}

// MarkNoKService sets the condition that the service is not ready
func (s *UiPathTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	UiPathCondSet.Manage(s).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *UiPathTargetStatus) IsReady() bool {
	return UiPathCondSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *UiPathTarget) GetConditionSet() apis.ConditionSet {
	return UiPathCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *UiPathTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
