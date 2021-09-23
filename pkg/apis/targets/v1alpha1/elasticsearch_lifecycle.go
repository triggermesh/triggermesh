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
	EventTypeElasticsearchStore    = "io.triggermesh.elasticsearch.doc.index"
	EventTypeElasticsearchResponse = "io.triggermesh.elasticsearch.doc.index.response"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*ElasticsearchTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeElasticsearchStore,
	}
}

// GetEventTypes implements EventSource.
func (*ElasticsearchTarget) GetEventTypes() []string {
	return []string{
		EventTypeElasticsearchResponse,
	}
}

// AsEventSource implements targets.EventSource.
func (s *ElasticsearchTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

// ElasticsearchCondSet is the group of possible conditions
var ElasticsearchCondSet = apis.NewLivingConditionSet(
	ConditionServiceReady,
	ConditionSecretsProvided,
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *ElasticsearchTargetStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return ElasticsearchCondSet.Manage(s).GetCondition(t)
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *ElasticsearchTargetStatus) InitializeConditions() {
	ElasticsearchCondSet.Manage(s).InitializeConditions()
	s.Address = &duckv1.Addressable{}
}

// GetGroupVersionKind returns the GroupVersionKind.
func (s *ElasticsearchTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ElasticsearchTarget")
}

// PropagateKServiceAvailability uses the availability of the provided KService to determine if
// ConditionServiceReady should be marked as true or false.
func (s *ElasticsearchTargetStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
	if ksvc.IsReady() && ksvc.Status.Address != nil && ksvc.Status.Address.URL != nil && !ksvc.Status.Address.URL.IsEmpty() {
		s.Address.URL = ksvc.Status.Address.URL
		ElasticsearchCondSet.Manage(s).MarkTrue(ConditionServiceReady)
		return
	} else if ksvc == nil {
		s.MarkNoKService(ReasonUnavailable, "Adapter service unknown: ksvc is not available")
	} else {
		s.MarkNoKService(ReasonUnavailable, "Adapter service \"%s/%s\" is unavailable", ksvc.Namespace, ksvc.Name)
	}
	s.Address.URL = nil
}

// MarkNoKService sets the condition that the service is not ready
func (s *ElasticsearchTargetStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
	ElasticsearchCondSet.Manage(s).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *ElasticsearchTargetStatus) IsReady() bool {
	return ElasticsearchCondSet.Manage(s).IsHappy()
}

// MarkSecrets sets the condition that the resource is valid
func (s *ElasticsearchTargetStatus) MarkSecrets() {
	ElasticsearchCondSet.Manage(s).MarkTrue(ConditionSecretsProvided)
}

// MarkNoSecrets sets the condition that the resource is not valid
func (s *ElasticsearchTargetStatus) MarkNoSecrets(reason, messageFormat string, messageA ...interface{}) {
	ElasticsearchCondSet.Manage(s).MarkFalse(ConditionSecretsProvided, reason, messageFormat, messageA...)
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *ElasticsearchTarget) GetConditionSet() apis.ConditionSet {
	return ElasticsearchCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *ElasticsearchTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
