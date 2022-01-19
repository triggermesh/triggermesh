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

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *Synchronizer) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Synchronizer")
}

// SynchronizerCondSet is the group of possible conditions
var SynchronizerCondSet = apis.NewLivingConditionSet(
	ConditionSinkProvided,
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *SynchronizerStatus) InitializeConditions() {
	SynchronizerCondSet.Manage(s).InitializeConditions()
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionDeployed should be marked as true or false.
func (s *SynchronizerStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		SynchronizerCondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		SynchronizerCondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	SynchronizerCondSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)

}

// MarkSink sets the SinkProvided condition to True using the given URI.
func (s *SynchronizerStatus) MarkSink(uri *apis.URL) {
	s.SinkURI = uri
	if uri == nil {
		SynchronizerCondSet.Manage(s).MarkFalse(ConditionSinkProvided,
			ReasonSinkEmpty, "The sink has no URI")
		return
	}
	SynchronizerCondSet.Manage(s).MarkTrue(ConditionSinkProvided)
}

// MarkNoSink sets the SinkProvided condition to False.
func (s *SynchronizerStatus) MarkNoSink() {
	s.SinkURI = nil
	SynchronizerCondSet.Manage(s).MarkFalse(ConditionSinkProvided,
		ReasonSinkNotFound, "The sink does not exist or its URI is not set")
}

// MarkNoKService sets the condition that the service is not ready
func (s *SynchronizerStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	SynchronizerCondSet.Manage(s).MarkFalse(ConditionDeployed, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *SynchronizerStatus) IsReady() bool {
	return SynchronizerCondSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *Synchronizer) GetConditionSet() apis.ConditionSet {
	return SynchronizerCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *Synchronizer) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
