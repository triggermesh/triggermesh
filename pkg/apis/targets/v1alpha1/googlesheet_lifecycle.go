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
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// GoogleSheetCondSet is the group of possible conditions.
var GoogleSheetCondSet = apis.NewLivingConditionSet(
	ConditionServiceReady,
	ConditionSecretsProvided,
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *GoogleSheetTargetStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return GoogleSheetCondSet.Manage(s).GetCondition(t)
}

// GetGroupVersionKind returns the GroupVersionKind.
func (*GoogleSheetTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GoogleSheetTarget")
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *GoogleSheetTargetStatus) InitializeConditions() {
	GoogleSheetCondSet.Manage(s).InitializeConditions()
	s.Address = &duckv1.Addressable{}
}

// Accepted event types
const (
	// EventTypeGoogleSheetAppend represents a task to append a row to a sheet.
	EventTypeGoogleSheetAppend = "io.triggermesh.googlesheet.append"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*GoogleSheetTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeGoogleSheetAppend,
	}
}

// GetEventTypes implements EventSource.
func (*GoogleSheetTarget) GetEventTypes() []string {
	return []string{
		EventTypeResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *GoogleSheetTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

// PropagateAvailability uses the readiness of the provided Knative Service to
// determine whether the ServiceReady condition should be marked as true or false.
func (s *GoogleSheetTargetStatus) PropagateAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		GoogleSheetCondSet.Manage(s).MarkUnknown(ConditionServiceReady, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		GoogleSheetCondSet.Manage(s).MarkTrue(ConditionServiceReady)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	GoogleSheetCondSet.Manage(s).MarkFalse(ConditionServiceReady, ReasonUnavailable, msg)
}

// MarkNoService sets the condition that the service is not ready.
func (s *GoogleSheetTargetStatus) MarkNoService(reason, messageFormat string, messageA ...interface{}) {
	GoogleSheetCondSet.Manage(s).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *GoogleSheetTargetStatus) IsReady() bool {
	return GoogleSheetCondSet.Manage(s).IsHappy()
}

// MarkSecrets sets the condition that the resource is valid.
func (s *GoogleSheetTargetStatus) MarkSecrets() {
	GoogleSheetCondSet.Manage(s).MarkTrue(ConditionSecretsProvided)
}

// MarkNoSecrets sets the condition that the resource is not valid.
func (s *GoogleSheetTargetStatus) MarkNoSecrets(err error) {
	GoogleSheetCondSet.Manage(s).MarkFalse(ConditionSecretsProvided,
		ReasonNotFound, err.Error())
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*GoogleSheetTarget) GetConditionSet() apis.ConditionSet {
	return GoogleSheetCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *GoogleSheetTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
