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

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// Managed event types
const (
	EventTypeSalesforceAPICall         = "io.triggermesh.salesforce.apicall"
	EventTypeSalesforceAPICallResponse = "io.triggermesh.salesforce.apicall.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*SalesforceTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeSalesforceAPICall,
	}
}

// GetEventTypes implements EventSource.
func (*SalesforceTarget) GetEventTypes() []string {
	return []string{
		EventTypeSalesforceAPICallResponse,
	}
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *SalesforceTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("SalesforceTarget")
}

// AsEventSource implements targets.EventSource.
func (s *SalesforceTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

var salesforceCondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *SalesforceTargetStatus) InitializeConditions() {
	salesforceCondSet.Manage(s).InitializeConditions()
}

// PropagateAvailability uses the readiness of the provided Knative Service to
// determine whether the Deployed condition should be marked as true or false.
func (s *SalesforceTargetStatus) PropagateAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		salesforceCondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		salesforceCondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	salesforceCondSet.Manage(s).MarkFalse(ConditionDeployed, ReasonUnavailable, msg)
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *SalesforceTarget) GetConditionSet() apis.ConditionSet {
	return salesforceCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *SalesforceTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
