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
	"context"
	"path"

	"go.uber.org/zap"

	appsv1 "k8s.io/api/apps/v1"
	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"knative.dev/eventing/pkg/apis/duck"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/triggermesh/triggermesh/pkg/status"
)

// Status conditions
const (
	// ConditionReady has status True when the component is ready to receive/send events.
	ConditionReady = apis.ConditionReady
	// ConditionSinkProvided has status True when the component has been configured with an event sink.
	ConditionSinkProvided apis.ConditionType = "SinkProvided"
	// ConditionDeployed has status True when the component's adapter is up and running.
	ConditionDeployed apis.ConditionType = "Deployed"
)

// Reasons for status conditions
const (
	// ReasonRBACNotBound is set on a Deployed condition when an adapter's
	// ServiceAccount cannot be bound.
	ReasonRBACNotBound = "RBACNotBound"
	// ReasonUnavailable is set on a Deployed condition when an adapter in unavailable.
	ReasonUnavailable = "AdapterUnavailable"

	// ReasonSinkNotFound is set on a SinkProvided condition when a sink does not exist.
	ReasonSinkNotFound = "SinkNotFound"
	// ReasonSinkEmpty is set on a SinkProvided condition when a sink URI is empty.
	ReasonSinkEmpty = "EmptySinkURI"
)

// DefaultConditionSet is a generic set of status conditions used by default in
// all components.
var DefaultConditionSet = NewConditionSet()

// NewConditionSet returns a set of status conditions for a component type.
// Default conditions can be augmented by passing condition types as function arguments.
func NewConditionSet(cts ...apis.ConditionType) apis.ConditionSet {
	return apis.NewLivingConditionSet(
		append(defaultConditionTypes, cts...)...,
	)
}

// defaultConditionTypes is a list of condition types common to all components.
var defaultConditionTypes = []apis.ConditionType{
	ConditionDeployed,
}

// EventSenderConditionSet is a set of conditions for instances that send
// events to a sink.
var EventSenderConditionSet = NewConditionSet(
	ConditionSinkProvided,
)

// Status defines the observed state of a component instance.
//
// +k8s:deepcopy-gen=true
type Status struct {
	duckv1.SourceStatus  `json:",inline"`
	duckv1.AddressStatus `json:",inline"`

	// Accepted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// CloudEventStatus contains attributes that event receivers can embed to
// declare the event types they accept.
//
// +k8s:deepcopy-gen=true
type CloudEventStatus struct {
	// AcceptedEventTypes are the CloudEvent types that a component can process.
	// +optional
	AcceptedEventTypes []string `json:"acceptedEventTypes,omitempty"`
}

// StatusManager manages the status of a TriggerMesh component.
type StatusManager struct {
	apis.ConditionSet
	*Status
}

// MarkSink sets the SinkProvided condition to True using the given URI.
func (m *StatusManager) MarkSink(uri *apis.URL) {
	m.SinkURI = uri

	if uri == nil {
		// A nil uri triggers a clearing of the stale SinkProvided status condition, providing that the sink
		// attribute is _optional_ for the given component.
		// If the sink attribute is non-optional for the component (e.g. all types in the 'sources' API group),
		// the SinkProvided status condition is expected to be "terminal" (always part of the ConditionSet
		// returned by the Reconcilable's GetConditionSet method), and ClearCondition will return an error.
		//
		// uri will only be nil in two cases:
		//  - the component type does not support sending events (does not implement EventSender)
		//  - the component type supports sending events, but this particular instance does not define a sink
		//    Destination (uses replies instead)
		if err := m.Manage(m).ClearCondition(ConditionSinkProvided); err != nil {
			m.Manage(m).MarkFalse(ConditionSinkProvided,
				ReasonSinkEmpty, "The sink has no URI")
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

// PropagateDeploymentAvailability uses the readiness of the provided
// Deployment to determine whether the Deployed condition should be marked as
// True or False.
// Given an optional PodInterface, the status of dependant Pods is inspected to
// generate a more meaningful failure reason in case of non-ready status of the
// Deployment.
func (m *StatusManager) PropagateDeploymentAvailability(ctx context.Context,
	d *appsv1.Deployment, pi coreclientv1.PodInterface) {

	// Deployments are not addressable
	m.Address = nil

	if d == nil {
		m.ConditionSet.Manage(m).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Deployment can not be determined")
		return
	}

	if duck.DeploymentIsAvailable(&d.Status, false) {
		m.ConditionSet.Manage(m).MarkTrue(ConditionDeployed)
		return
	}

	reason := ReasonUnavailable
	msg := "The adapter Deployment is unavailable"

	for _, cond := range d.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable && cond.Message != "" {
			msg += ": " + cond.Message
		}
	}

	if pi != nil {
		ws, err := status.DeploymentPodsWaitingState(d, pi)
		if err != nil {
			logging.FromContext(ctx).Warn("Unable to look up statuses of dependant Pods", zap.Error(err))
		} else if ws != nil {
			reason = status.ExactReason(ws)
			msg += ": " + ws.Message
		}
	}

	m.ConditionSet.Manage(m).MarkFalse(ConditionDeployed, reason, msg)
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
