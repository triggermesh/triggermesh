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
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// Managed event types
const (
	EventType$TARGETFULLCASE GenericResponse = "io.triggermesh.$TARGET.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*$TARGETFULLCASE Target) AcceptedEventTypes() []string {
	return []string{
		"*",
	}
}

// GetEventTypes implements EventSource.
func (*$TARGETFULLCASE Target) GetEventTypes() []string {
	return []string{
		EventType$TARGETFULLCASE GenericResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *$TARGETFULLCASE Target) AsEventSource() string {
	return "https://" + "SOMETHINGUSEFULE"
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *$TARGETFULLCASE Target) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("$TARGETFULLCASE Target")
}

// $TARGETFULLCASE CondSet is the group of possible conditions
var $TARGETFULLCASE CondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *$TARGETFULLCASE TargetStatus) InitializeConditions() {
	$TARGETFULLCASE CondSet.Manage(s).InitializeConditions()
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionDeployed should be marked as true or false.
func (s *$TARGETFULLCASE TargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		$TARGETFULLCASE CondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		$TARGETFULLCASE CondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	$TARGETFULLCASE CondSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)

}

// MarkNoKService sets the condition that the service is not ready
func (s *$TARGETFULLCASE TargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	$TARGETFULLCASE CondSet.Manage(s).MarkFalse(ConditionDeployed, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *$TARGETFULLCASE TargetStatus) IsReady() bool {
	return $TARGETFULLCASE CondSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *$TARGETFULLCASE Target) GetConditionSet() apis.ConditionSet {
	return $TARGETFULLCASE CondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *$TARGETFULLCASE Target) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
