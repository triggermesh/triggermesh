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
	EventType{{.UppercaseName}}GenericResponse = "io.triggermesh.{{.Name}}.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*{{.UppercaseName}}) AcceptedEventTypes() []string {
	return []string{
		"*",
	}
}

// GetEventTypes implements EventSource.
func (*{{.UppercaseName}}) GetEventTypes() []string {
	return []string{
		EventType{{.UppercaseName}}GenericResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *{{.UppercaseName}}) AsEventSource() string {
	return "https://" + "SOMETHINGUSEFULE"
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *{{.UppercaseName}}) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("{{.UppercaseName}} Target")
}

// {{.UppercaseName}}CondSet is the group of possible conditions
var {{.UppercaseName}}CondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *{{.UppercaseName}}Status) InitializeConditions() {
	{{.UppercaseName}}CondSet.Manage(s).InitializeConditions()
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionDeployed should be marked as true or false.
func (s *{{.UppercaseName}}Status) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		{{.UppercaseName}}CondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		{{.UppercaseName}}CondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	{{.UppercaseName}}CondSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)

}

// MarkNoKService sets the condition that the service is not ready
func (s *{{.UppercaseName}}Status) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	{{.UppercaseName}}CondSet.Manage(s).MarkFalse(ConditionDeployed, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *{{.UppercaseName}}Status) IsReady() bool {
	return {{.UppercaseName}}CondSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *{{.UppercaseName}}) GetConditionSet() apis.ConditionSet {
	return {{.UppercaseName}}CondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *{{.UppercaseName}}) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
