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

// ZendeskCondSet is the group of possible conditions
var ZendeskCondSet = apis.NewLivingConditionSet(
	ConditionServiceReady,
	ConditionSecretsProvided,
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *ZendeskTargetStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return ZendeskCondSet.Manage(s).GetCondition(t)
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *ZendeskTargetStatus) InitializeConditions() {
	ZendeskCondSet.Manage(s).InitializeConditions()
	s.Address = &duckv1.Addressable{}
}

// GetGroupVersionKind returns the GroupVersionKind.
func (s *ZendeskTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ZendeskTarget")
}

// Accepted event types
const (
	// EventTypeZendesk represents a task to create a Zendesk ticket.
	EventTypeZendeskTicketCreate = "com.zendesk.ticket.create"
	EventTypeZendeskTagCreate    = "com.zendesk.ticket.tag.add"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*ZendeskTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeZendeskTicketCreate,
		EventTypeZendeskTagCreate,
	}
}

// GetEventTypes implements EventSource.
func (*ZendeskTarget) GetEventTypes() []string {
	return []string{
		EventTypeResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *ZendeskTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionServiceReady should be marked as true or false.
func (s *ZendeskTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc != nil && ksvc.IsReady() {
		s.Address = ksvc.Status.Address
		ZendeskCondSet.Manage(s).MarkTrue(ConditionServiceReady)
		return
	} else if ksvc == nil {
		s.MarkNoKService(ReasonUnavailable, "Adapter service unknown: ksvc is not available")
	} else {
		s.MarkNoKService(ReasonUnavailable, "Adapter service \"%s/%s\" is unavailable", ksvc.Namespace, ksvc.Name)
	}
	s.Address.URL = nil
}

// MarkNoKService sets the condition that the service is not ready
func (s *ZendeskTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	ZendeskCondSet.Manage(s).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *ZendeskTargetStatus) IsReady() bool {
	return ZendeskCondSet.Manage(s).IsHappy()
}

// MarkSecrets sets the condition that the resource is valid
func (s *ZendeskTargetStatus) MarkSecrets() {
	ZendeskCondSet.Manage(s).MarkTrue(ConditionSecretsProvided)
}

// MarkNoSecrets sets the condition that the resource is not valid
func (s *ZendeskTargetStatus) MarkNoSecrets(reason, messageFormat string, messageA ...interface{}) {
	ZendeskCondSet.Manage(s).MarkFalse(ConditionSecretsProvided, reason, messageFormat, messageA...)
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *ZendeskTarget) GetConditionSet() apis.ConditionSet {
	return ZendeskCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *ZendeskTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
