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
	EventType{{.Kind}}GenericResponse = "io.triggermesh.{{.LowercaseKind}}.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*{{.Kind}}) AcceptedEventTypes() []string {
	return []string{
		"*",
	}
}

// GetEventTypes implements EventSource.
func (*{{.Kind}}) GetEventTypes() []string {
	return []string{
		EventType{{.Kind}}GenericResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *{{.Kind}}) AsEventSource() string {
	return "https://" + "SOMETHINGUSEFULE"
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *{{.Kind}}) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("{{.Kind}} Target")
}

// {{.Kind}}CondSet is the group of possible conditions
var {{.Kind}}CondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *{{.Kind}}Status) InitializeConditions() {
	{{.Kind}}CondSet.Manage(s).InitializeConditions()
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionDeployed should be marked as true or false.
func (s *{{.Kind}}Status) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		{{.Kind}}CondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		{{.Kind}}CondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	{{.Kind}}CondSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)

}

// MarkNoKService sets the condition that the service is not ready
func (s *{{.Kind}}Status) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	{{.Kind}}CondSet.Manage(s).MarkFalse(ConditionDeployed, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *{{.Kind}}Status) IsReady() bool {
	return {{.Kind}}CondSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *{{.Kind}}) GetConditionSet() apis.ConditionSet {
	return {{.Kind}}CondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *{{.Kind}}) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
