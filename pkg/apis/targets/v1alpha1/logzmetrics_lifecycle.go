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

// Reasons for status conditions
const (
	// LogzMetricsReasonWrongSpec is set when an adapter cannot be built from the spec.
	LogzMetricsReasonWrongSpec = "WrongSpec"
)

// Managed event types
const (
	EventTypeOpenTelemetryMetricsPush = "io.triggermesh.opentelemetry.metrics.push"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*LogzMetricsTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeOpenTelemetryMetricsPush,
	}
}

// GetEventTypes implements EventSource.
func (*LogzMetricsTarget) GetEventTypes() []string {
	return []string{}
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *LogzMetricsTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("LogzMetricsTarget")
}

// AsEventSource implements targets.EventSource.
func (s *LogzMetricsTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

var logzmetricsConditionSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *LogzMetricsTargetStatus) InitializeConditions() {
	logzmetricsConditionSet.Manage(s).InitializeConditions()
}

// PropagateAvailability uses the readiness of the provided Knative Service to
// determine whether the Deployed condition should be marked as true or false.
func (s *LogzMetricsTargetStatus) PropagateAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		logzmetricsConditionSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		logzmetricsConditionSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	logzmetricsConditionSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)
}

// MarkNotDeployed sets the condition that the service has not been deployed.
func (s *LogzMetricsTargetStatus) MarkNotDeployed(reason, messageFormat string, messageA ...interface{}) {
	logzmetricsConditionSet.Manage(s).MarkFalse(ConditionDeployed, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *LogzMetricsTargetStatus) IsReady() bool {
	return logzmetricsConditionSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *LogzMetricsTarget) GetConditionSet() apis.ConditionSet {
	return logzmetricsConditionSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *LogzMetricsTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
