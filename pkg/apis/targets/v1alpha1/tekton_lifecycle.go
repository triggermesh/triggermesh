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

// TektonCondSet is the group of possible conditions
var TektonCondSet = apis.NewLivingConditionSet(
	ConditionServiceReady,
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *TektonTargetStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return TektonCondSet.Manage(s).GetCondition(t)
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *TektonTargetStatus) InitializeConditions() {
	TektonCondSet.Manage(s).InitializeConditions()
	s.Address = &duckv1.Addressable{}
}

// GetGroupVersionKind returns the GroupVersionKind.
func (*TektonTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TektonTarget")
}

// Accepted event types
const (
	// EventTypeTektonRun represents a task to run a Task or Pipeline.
	EventTypeTektonRun = "io.triggermesh.tekton.run"
	// EventTypeTektonReap event to trigger reaping of completed runs
	EventTypeTektonReap = "io.triggermesh.tekton.reap"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*TektonTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeTektonRun,
		EventTypeTektonReap,
	}
}

// GetEventTypes implements EventSource.
func (*TektonTarget) GetEventTypes() []string {
	return []string{
		EventTypeResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (t *TektonTarget) AsEventSource() string {
	kind := strings.ToLower(t.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + t.Namespace + "." + t.Name
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionServiceReady should be marked as true or false.
func (s *TektonTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc.IsReady() && ksvc.Status.Address != nil && ksvc.Status.Address.URL != nil && !ksvc.Status.Address.URL.IsEmpty() {
		s.Address.URL = ksvc.Status.Address.URL
		TektonCondSet.Manage(s).MarkTrue(ConditionServiceReady)
		return
	} else if ksvc == nil {
		s.MarkNoKService(ReasonUnavailable, "Adapter service unknown: ksvc is not available")
	} else {
		s.MarkNoKService(ReasonUnavailable, "Adapter service \"%s/%s\" is unavailable", ksvc.Namespace, ksvc.Name)
	}
	s.Address.URL = nil
}

// MarkNoKService sets the condition that the service is not ready
func (s *TektonTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	TektonCondSet.Manage(s).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *TektonTargetStatus) IsReady() bool {
	return TektonCondSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*TektonTarget) GetConditionSet() apis.ConditionSet {
	return TektonCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (t *TektonTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}
