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
	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// Managed event types
const (
	EventTypeJiraIssueCreate = "io.triggermesh.jira.issue.create"
	EventTypeJiraIssueGet    = "io.triggermesh.jira.issue.get"
	EventTypeJiraCustom      = "io.triggermesh.jira.custom"

	EventTypeJiraIssue          = "io.triggermesh.jira.issue"
	EventTypeJiraCustomResponse = "io.triggermesh.jira.custom.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*JiraTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeJiraIssueCreate,
		EventTypeJiraIssueGet,
		EventTypeJiraCustom,
	}
}

// GetEventTypes implements EventSource.
func (*JiraTarget) GetEventTypes() []string {
	return []string{
		EventTypeJiraIssue,
		EventTypeJiraCustomResponse,
	}
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *JiraTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("JiraTarget")
}

var jiraCondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *JiraTargetStatus) InitializeConditions() {
	jiraCondSet.Manage(s).InitializeConditions()
}

// PropagateAvailability uses the readiness of the provided Knative Service to
// determine whether the Deployed condition should be marked as true or false.
func (s *JiraTargetStatus) PropagateAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		jiraCondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		jiraCondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	jiraCondSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *JiraTarget) GetConditionSet() apis.ConditionSet {
	return jiraCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *JiraTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
