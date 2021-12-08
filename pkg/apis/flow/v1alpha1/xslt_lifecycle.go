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

// Reasons for status conditions
const (
	// XSLTTransformReasonWrongSpec is set when an adapter cannot be built from the spec.
	XSLTTransformReasonWrongSpec = "WrongSpec"
)

const (
	// ConditionReady is set when the runtime resources for the component
	// are ready to be used.
	XSLTTransformConditionReady = apis.ConditionReady
)

// Managed event types
const (
	EventTypeXSLTTransform = "io.triggermesh.xslt.transform"
)

// GetGroupVersionKind returns the GroupVersionKind.
func (*XSLTTransform) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("XSLTTransform")
}

var xsltTransformrCondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*XSLTTransform) GetConditionSet() apis.ConditionSet {
	return xsltTransformrCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (o *XSLTTransform) GetStatus() *duckv1.Status {
	return &o.Status.Status
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *XSLTTransformStatus) InitializeConditions() {
	xsltTransformrCondSet.Manage(s).InitializeConditions()
}

// PropagateAvailability uses the readiness of the provided Knative Service to
// determine whether the Deployed condition should be marked as true or false.
func (s *XSLTTransformStatus) PropagateAvailability(ksvc *servingv1.Service) {
	if ksvc == nil {
		xsltTransformrCondSet.Manage(s).MarkUnknown(ConditionDeployed, ReasonUnavailable,
			"The status of the adapter Service can not be determined")
		return
	}

	if s.Address == nil {
		s.Address = &duckv1.Addressable{}
	}
	s.Address.URL = ksvc.Status.URL

	if ksvc.IsReady() {
		xsltTransformrCondSet.Manage(s).MarkTrue(ConditionDeployed)
		return
	}

	msg := "The adapter Service is unavailable"
	readyCond := ksvc.Status.GetCondition(servingv1.ServiceConditionReady)
	if readyCond != nil && readyCond.Message != "" {
		msg += ": " + readyCond.Message
	}

	s.MarkNotDeployed(ReasonUnavailable, msg)
}

// MarkNotDeployed sets the condition that the service has not been deployed.
func (s *XSLTTransformStatus) MarkNotDeployed(reason, messageFormat string, messageA ...interface{}) {
	xsltTransformrCondSet.Manage(s).MarkFalse(ConditionDeployed, reason, messageFormat, messageA...)
}
