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
)

// GetGroupVersionKind returns the GroupVersionKind.
func (x *XsltTransform) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("XsltTransform")
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (x *XsltTransform) GetConditionSet() apis.ConditionSet {
	return eventFlowConditionSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (x *XsltTransform) GetStatus() *duckv1.Status {
	return &x.Status.Status
}

// GetStatusManager implements EventFlowComponent.
func (x *XsltTransform) GetStatusManager() *EventFlowStatusManager {
	return &EventFlowStatusManager{
		ConditionSet:    x.GetConditionSet(),
		EventFlowStatus: &x.Status,
	}
}

// AsEventSource implements EventFlowComponent.
func (s *XsltTransform) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}

// // XsltTransformCondSet is the group of possible conditions
// var XsltTransformCondSet = apis.NewLivingConditionSet(
// 	ConditionServiceReady,
// 	ConditionSecretsProvided,
// )

// // GetCondition returns the condition currently associated with the given type, or nil.
// func (s *XsltTransformStatus) GetCondition(t apis.ConditionType) *apis.Condition {
// 	return XsltTransformCondSet.Manage(s).GetCondition(t)
// }

// // InitializeConditions sets relevant unset conditions to Unknown state.
// func (s *XsltTransformStatus) InitializeConditions() {
// 	XsltTransformCondSet.Manage(s).InitializeConditions()
// 	s.Address = &duckv1.Addressable{}
// }

// // PropagateKServiceAvailability uses the availability of the provided KService to determine if
// // ConditionServiceReady should be marked as true or false.
// func (s *XsltTransformStatus) PropagateKServiceAvailability(ksvc *servingv1.Service) {
// 	if ksvc != nil && ksvc.IsReady() {
// 		s.Address = ksvc.Status.Address
// 		XsltTransformCondSet.Manage(s).MarkTrue(ConditionServiceReady)
// 		return
// 	} else if ksvc == nil {
// 		s.MarkNoKService(ReasonUnavailable, "Adapter service unknown: ksvc is not available")
// 	} else {
// 		s.MarkNoKService(ReasonUnavailable, "Adapter service \"%s/%s\" is unavailable", ksvc.Namespace, ksvc.Name)
// 	}
// 	s.Address.URL = nil
// }

// // MarkNoKService sets the condition that the service is not ready
// func (s *XsltTransformStatus) MarkNoKService(reason, messageFormat string, messageA ...interface{}) {
// 	XsltTransformCondSet.Manage(s).MarkFalse(ConditionServiceReady, reason, messageFormat, messageA...)
// }

// // IsReady returns true if the resource is ready overall.
// func (s *XsltTransformStatus) IsReady() bool {
// 	return XsltTransformCondSet.Manage(s).IsHappy()
// }
