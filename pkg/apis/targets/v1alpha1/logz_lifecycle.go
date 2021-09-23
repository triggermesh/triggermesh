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

// Managed event types
const (
	EventTypeLogzShipResponse = "io.triggermesh.logz.ship.response"
)

// GetEventTypes implements EventSource.
func (*LogzTarget) GetEventTypes() []string {
	return []string{
		EventTypeLogzShipResponse,
	}
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *LogzTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("LogzTarget")
}

// AsEventSource implements targets.EventSource.
func (s *LogzTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

var logzConditionSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *LogzTargetStatus) InitializeConditions() {
	logzConditionSet.Manage(s).InitializeConditions()
}

// PropagateAvailability uses the readiness of the provided Knative Service to
// determine whether the Deployed condition should be marked as true or false.
func (s *LogzTargetStatus) PropagateAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		logzConditionSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		logzConditionSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	logzConditionSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)
}

// MarkNoKService sets the condition that the service is not ready
func (s *LogzTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	logzConditionSet.Manage(s).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *LogzTargetStatus) IsReady() bool {
	return logzConditionSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *LogzTarget) GetConditionSet() apis.ConditionSet {
	return logzConditionSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *LogzTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
