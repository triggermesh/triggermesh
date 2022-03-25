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
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/targets/status"
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

// eventSenderConditionSet is a set of conditions for instances that send
// events to a sink.
var eventSenderConditionSet = NewTargetConditionSet(
	ConditionSinkProvided,
)

// StatusManager manages the status of a TriggerMesh component.
//
// +k8s:deepcopy-gen=false
type StatusManager struct {
	apis.ConditionSet
	*TargetStatus
}

// MarkSink sets the SinkProvided condition to True using the given URI.
func (m *StatusManager) MarkSink(uri *apis.URL) {
	m.SinkURI = uri
	if uri == nil {
		// A nil uri triggers a clearing of the stale SinkProvided status condition.
		//
		// uri will only be nil in two cases:
		//  - the component type does not support sending events (does not implement EventSender)
		//  - the component type supports sending events, but this particular instance does not define a sink
		//    Destination (uses replies instead)
		//
		// In any of these two cases, the ConditionSet will not include the SinkProvided status condition,
		// therefore ClearCondition will never return an error. If it does, it would be an indicator of a
		// wrongly implemented GetConditionSet method (duckv1.KRShaped interface), in which case we panic
		// intentionally so that the issue gets caught early by unit tests.
		if err := m.Manage(m).ClearCondition(ConditionSinkProvided); err != nil {
			panic(fmt.Errorf("unexpected failure clearing the %q status condition: %w",
				ConditionSinkProvided, err))
		}

		return
	}
	m.Manage(m).MarkTrue(ConditionSinkProvided)
}

// MarkNoSink sets the SinkProvided condition to False.
func (m *StatusManager) MarkNoSink() {
	m.SinkURI = nil
	m.ConditionSet.Manage(m).MarkFalse(ConditionSinkProvided,
		ReasonSinkNotFound, "The sink does not exist or its URI is not set")
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

// IsInformed returns if the value is informed in any of the available choices.
func (v *ValueFromField) IsInformed() bool {
	if v != nil &&
		(v.Value != "" ||
			v.ValueFromSecret != nil && v.ValueFromSecret.Name != "" && v.ValueFromSecret.Key != "" ||
			v.ValueFromConfigMap != nil && v.ValueFromConfigMap.Name != "" && v.ValueFromConfigMap.Key != "") {
		return true
	}

	return false
}

// ToEnvironmentVariable returns a kubernetes environment variable from
// a ValueFromField.
func (v *ValueFromField) ToEnvironmentVariable(name string) *corev1.EnvVar {
	env := &corev1.EnvVar{
		Name: name,
	}

	switch {
	case v == nil:

	case v.Value != "":
		env.Value = v.Value

	case v.ValueFromSecret != nil && v.ValueFromSecret.Name != "" && v.ValueFromSecret.Key != "":
		env.ValueFrom = &corev1.EnvVarSource{
			SecretKeyRef: v.ValueFromSecret,
		}

	case v.ValueFromConfigMap != nil && v.ValueFromConfigMap.Name != "" && v.ValueFromConfigMap.Key != "":
		env.ValueFrom = &corev1.EnvVarSource{
			ConfigMapKeyRef: v.ValueFromConfigMap,
		}
	}

	return env
}
