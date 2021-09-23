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

// TwilioCondSet is the group of possible conditions
var TwilioCondSet = apis.NewLivingConditionSet(
	ConditionServiceReady,
	ConditionSecretsProvided,
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *TwilioTargetStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return TwilioCondSet.Manage(s).GetCondition(t)
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *TwilioTargetStatus) InitializeConditions() {
	TwilioCondSet.Manage(s).InitializeConditions()
	s.Address = &duckv1.Addressable{}
}

// GetGroupVersionKind returns the GroupVersionKind.
func (s *TwilioTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TwilioTarget")
}

// Accepted event types
const (
	// EventTypeTwilioSMSSend represents a task to send a SMS.
	EventTypeTwilioSMSSend = "io.triggermesh.twilio.sms.send"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*TwilioTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeTwilioSMSSend,
	}
}

// GetEventTypes implements EventSource.
func (*TwilioTarget) GetEventTypes() []string {
	return []string{
		EventTypeResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *TwilioTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionServiceReady should be marked as true or false.
func (s *TwilioTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc.IsReady() && ksvc.Status.Address != nil && ksvc.Status.Address.URL != nil && !ksvc.Status.Address.URL.IsEmpty() {
		s.Address.URL = ksvc.Status.Address.URL
		TwilioCondSet.Manage(s).MarkTrue(ConditionServiceReady)
		return
	} else if ksvc == nil {
		s.MarkNoKService(ReasonUnavailable, "Adapter service unknown: ksvc is not available")
	} else {
		s.MarkNoKService(ReasonUnavailable, "Adapter service \"%s/%s\" is unavailable", ksvc.Namespace, ksvc.Name)
	}
	s.Address.URL = nil
}

// MarkNoKService sets the condition that the service is not ready
func (s *TwilioTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	TwilioCondSet.Manage(s).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *TwilioTargetStatus) IsReady() bool {
	return TwilioCondSet.Manage(s).IsHappy()
}

// MarkSecrets sets the condition that the resource is valid
func (s *TwilioTargetStatus) MarkSecrets() {
	TwilioCondSet.Manage(s).MarkTrue(ConditionSecretsProvided)
}

// MarkNoSecrets sets the condition that the resource is not valid
func (s *TwilioTargetStatus) MarkNoSecrets(reason, messageFormat string, messageA ...interface{}) {
	TwilioCondSet.Manage(s).MarkFalse(ConditionSecretsProvided, reason, messageFormat, messageA...)
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *TwilioTarget) GetConditionSet() apis.ConditionSet {
	return TwilioCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *TwilioTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
