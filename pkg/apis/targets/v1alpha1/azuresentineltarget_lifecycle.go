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
	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// Managed event types
const (
	EventTypeAzureSentinelTargetGenericResponse = "io.triggermesh.azuresentineltarget.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*AzureSentinelTarget) AcceptedEventTypes() []string {
	return []string{
		"*",
	}
}

// GetEventTypes implements EventSource.
func (*AzureSentinelTarget) GetEventTypes() []string {
	return []string{
		EventTypeAzureSentinelTargetGenericResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *AzureSentinelTarget) AsEventSource() string {
	return "https://" + "REPLACEME"
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *AzureSentinelTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AzureSentinelTarget Target")
}

// AzureSentinelTargetCondSet is the group of possible conditions
var AzureSentinelTargetCondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *AzureSentinelTargetStatus) InitializeConditions() {
	AzureSentinelTargetCondSet.Manage(s).InitializeConditions()
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionDeployed should be marked as true or false.
func (s *AzureSentinelTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		AzureSentinelTargetCondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		AzureSentinelTargetCondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	AzureSentinelTargetCondSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)

}

// MarkNoKService sets the condition that the service is not ready
func (s *AzureSentinelTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	AzureSentinelTargetCondSet.Manage(s).MarkFalse(ConditionDeployed, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *AzureSentinelTargetStatus) IsReady() bool {
	return AzureSentinelTargetCondSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *AzureSentinelTarget) GetConditionSet() apis.ConditionSet {
	return AzureSentinelTargetCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *AzureSentinelTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
