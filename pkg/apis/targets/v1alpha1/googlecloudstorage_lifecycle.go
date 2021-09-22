/*
Copyright (c) 2021 TriggerMesh Inc.

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
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// Managed event types
const (
	EventTypeGoogleCloudStorageObjectInsert = "com.google.cloud.storage.object.insert"

	EventTypeGoogleCloudStorageResponse = "com.google.cloud.storage.object.insert.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*GoogleCloudStorageTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeGoogleCloudStorageObjectInsert,
		EventTypeWildcard,
	}
}

// GetEventTypes implements EventSource.
func (*GoogleCloudStorageTarget) GetEventTypes() []string {
	return []string{
		EventTypeGoogleCloudStorageResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *GoogleCloudStorageTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *GoogleCloudStorageTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GoogleCloudStorageTarget")
}

// GoogleCloudStorageCondSet is the group of possible conditions
var GoogleCloudStorageCondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *GoogleCloudStorageTargetStatus) InitializeConditions() {
	GoogleCloudStorageCondSet.Manage(s).InitializeConditions()
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionServiceReady should be marked as true or false.
func (s *GoogleCloudStorageTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		GoogleSheetCondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		GoogleCloudStorageCondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	GoogleCloudStorageCondSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)
}

// MarkNoKService sets the condition that the service is not ready
func (s *GoogleCloudStorageTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	GoogleCloudStorageCondSet.Manage(s).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *GoogleCloudStorageTargetStatus) IsReady() bool {
	return GoogleCloudStorageCondSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *GoogleCloudStorageTarget) GetConditionSet() apis.ConditionSet {
	return GoogleCloudStorageCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *GoogleCloudStorageTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
