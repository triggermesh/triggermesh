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

// SendgridCondSet is the group of possible conditions
var SendgridCondSet = apis.NewLivingConditionSet(
	ConditionServiceReady,
	ConditionSecretsProvided,
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *SendGridTargetStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return SendgridCondSet.Manage(s).GetCondition(t)
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *SendGridTargetStatus) InitializeConditions() {
	SendgridCondSet.Manage(s).InitializeConditions()
	s.Address = &duckv1.Addressable{}
}

// GetGroupVersionKind returns the GroupVersionKind.
func (s *SendGridTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("SendGridTarget")
}

// Accepted event types
const (
	// EventTypeSendGridEmailSend represents a task to send an email.
	EventTypeSendGridEmailSend = "io.triggermesh.sendgrid.email.send"
	// EventTypeSendGridEmailSendResponse represents a response from the API after sending an email
	EventTypeSendGridEmailSendResponse = "io.triggermesh.sendgrid.email.send.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*SendGridTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeSendGridEmailSend,
		EventTypeWildcard,
	}
}

// GetEventTypes implements EventSource.
func (*SendGridTarget) GetEventTypes() []string {
	return []string{
		EventTypeResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *SendGridTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionServiceReady should be marked as true or false.
func (s *SendGridTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc.IsReady() && ksvc.Status.Address != nil && ksvc.Status.Address.URL != nil && !ksvc.Status.Address.URL.IsEmpty() {
		s.Address.URL = ksvc.Status.Address.URL
		SendgridCondSet.Manage(s).MarkTrue(ConditionServiceReady)
		return
	}
	s.Address.URL = nil
	s.MarkNoKService(ReasonUnavailable, "Adapter service \"%s/%s\" is unavailable", ksvc.Namespace, ksvc.Name)
}

// MarkNoKService sets the condition that the service is not ready
func (s *SendGridTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	SendgridCondSet.Manage(s).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *SendGridTargetStatus) IsReady() bool {
	return SendgridCondSet.Manage(s).IsHappy()
}

// MarkSecrets sets the condition that the resource is valid
func (s *SendGridTargetStatus) MarkSecrets() {
	SendgridCondSet.Manage(s).MarkTrue(ConditionSecretsProvided)
}

// MarkNoSecrets sets the condition that the resource is not valid
func (s *SendGridTargetStatus) MarkNoSecrets(reason, messageFormat string, messageA ...interface{}) {
	SendgridCondSet.Manage(s).MarkFalse(ConditionSecretsProvided, reason, messageFormat, messageA...)
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *SendGridTarget) GetConditionSet() apis.ConditionSet {
	return SendgridCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *SendGridTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
