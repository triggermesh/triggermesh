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

// Managed event types
const (
	EventTypeGoogleCloudWorkflowsRun = "io.trigermesh.google.workflows.run"

	EventTypeGoogleCloudWorkflowsRunResponse = "io.trigermesh.google.workflows.run.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*GoogleCloudWorkflowsTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeGoogleCloudWorkflowsRun,
	}
}

// GetEventTypes implements EventSource.
func (*GoogleCloudWorkflowsTarget) GetEventTypes() []string {
	return []string{
		EventTypeGoogleCloudWorkflowsRunResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *GoogleCloudWorkflowsTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *GoogleCloudWorkflowsTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GoogleCloudWorkflowsTarget")
}

// GoogleCloudWorkflowsCondSet is the group of possible conditions
var GoogleCloudWorkflowsCondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *GoogleCloudWorkflowsTargetStatus) InitializeConditions() {
	GoogleCloudWorkflowsCondSet.Manage(s).InitializeConditions()
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionDeployed should be marked as true or false.
func (s *GoogleCloudWorkflowsTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		GoogleCloudWorkflowsCondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		GoogleCloudWorkflowsCondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	GoogleCloudWorkflowsCondSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)

}

// MarkNoKService sets the condition that the service is not ready
func (s *GoogleCloudWorkflowsTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	GoogleCloudWorkflowsCondSet.Manage(s).MarkFalse(ConditionDeployed, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *GoogleCloudWorkflowsTargetStatus) IsReady() bool {
	return GoogleCloudWorkflowsCondSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *GoogleCloudWorkflowsTarget) GetConditionSet() apis.ConditionSet {
	return GoogleCloudWorkflowsCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *GoogleCloudWorkflowsTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
