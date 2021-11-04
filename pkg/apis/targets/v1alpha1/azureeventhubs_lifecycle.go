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
	EventTypeAzureEventHubsGenericResponse = "io.triggermesh.azure.eventhubs.put.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*AzureEventHubsTarget) AcceptedEventTypes() []string {
	return []string{
		"*",
	}
}

// GetEventTypes implements EventSource.
func (*AzureEventHubsTarget) GetEventTypes() []string {
	return []string{
		EventTypeAzureEventHubsGenericResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *AzureEventHubsTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *AzureEventHubsTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AzureEventHubsTarget")
}

// AzureEventHubsCondSet is the group of possible conditions
var AzureEventHubsCondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *AzureEventHubsTargetStatus) InitializeConditions() {
	AzureEventHubsCondSet.Manage(s).InitializeConditions()
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionDeployed should be marked as true or false.
func (s *AzureEventHubsTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		AzureEventHubsCondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		AzureEventHubsCondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	AzureEventHubsCondSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)

}

// MarkNoKService sets the condition that the service is not ready
func (s *AzureEventHubsTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	AzureEventHubsCondSet.Manage(s).MarkFalse(ConditionDeployed, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *AzureEventHubsTargetStatus) IsReady() bool {
	return AzureEventHubsCondSet.Manage(s).IsHappy()
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *AzureEventHubsTarget) GetConditionSet() apis.ConditionSet {
	return AzureEventHubsCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *AzureEventHubsTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
