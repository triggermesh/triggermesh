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
	"path"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/targets/status"
)

// Accepted event types
const (
	// EventTypeWildcard represents events of any type.
	EventTypeWildcard = "*"
)

// Returned event types
const (
	// EventTypeResponse is the event type of a generic response returned by a target.
	EventTypeResponse = "io.triggermesh.targets.response"
)

// targetConditionSet is a generic set of status conditions used by
// default in all targets.
var targetConditionSet = NewTargetConditionSet()

// NewTargetConditionSet returns a set of status conditions for a target.
// Default conditions can be augmented by passing condition types as function
// arguments.
func NewTargetConditionSet(cts ...apis.ConditionType) apis.ConditionSet {
	return apis.NewLivingConditionSet(
		append(targetConditionTypes, cts...)...,
	)
}

// targetConditionTypes is a list of condition types common to all targets.
var targetConditionTypes = []apis.ConditionType{
	ConditionDeployed,
}

// StatusManager manages the status of a TriggerMesh component.
//
// +k8s:deepcopy-gen=false
type StatusManager struct {
	apis.ConditionSet
	*TargetStatus
}

// MarkRBACNotBound sets the Deployed condition to False, indicating that the
// adapter's ServiceAccount couldn't be bound.
func (m *StatusManager) MarkRBACNotBound() {
	m.ConditionSet.Manage(m).MarkFalse(ConditionDeployed,
		ReasonRBACNotBound, "The adapter's ServiceAccount can not be bound")
}

// PropagateServiceAvailability uses the readiness of the provided Service to
// determine whether the Deployed condition should be marked as True or False.
func (m *StatusManager) PropagateServiceAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		m.ConditionSet.Manage(m).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if m.Address == nil {
		m.Address = &duckv1.Addressable{}
	}
	m.Address.URL = ksvc.Status.URL.DeepCopy()

	if ksvc.IsReady() {
		m.ConditionSet.Manage(m).MarkTrue(ConditionDeployed)
		return
	}

	reason := ReasonUnavailable
	msg := "The adapter Service is unavailable"

	// the RoutesReady condition surfaces the reason why network traffic
	// cannot be routed to the Service
	routesCond := ksvc.Status.GetCondition(servingv1.ServiceConditionRoutesReady)
	if routesCond != nil && routesCond.Message != "" {
		msg += "; " + routesCond.Message
	}

	// the ConfigurationsReady condition surfaces the reason why an
	// underlying Pod is failing
	configCond := ksvc.Status.GetCondition(servingv1.ServiceConditionConfigurationsReady)
	if configCond != nil && configCond.Message != "" {
		if r := status.ExactReason(configCond); r != configCond.Reason {
			reason = r
		}
		msg += "; " + configCond.Message
	}

	m.ConditionSet.Manage(m).MarkFalse(ConditionDeployed, reason, msg)
}

// SetRoute appends the given URL path to the current target's URL.
func (m *StatusManager) SetRoute(urlPath string) {
	if m.Address == nil || m.Address.URL == nil {
		return
	}

	m.Address.URL.Path = path.Join(m.Address.URL.Path, urlPath)
}
