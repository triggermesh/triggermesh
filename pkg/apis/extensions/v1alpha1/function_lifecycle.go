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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

var condSet = apis.NewLivingConditionSet(
	conditionReady,
	conditionSinkReady,
	conditionServiceReady,
	conditionConfigmapReady,
)

const (
	conditionReady = apis.ConditionReady

	conditionSinkReady      apis.ConditionType = "SinkReady"
	conditionServiceReady   apis.ConditionType = "ServiceReady"
	conditionConfigmapReady apis.ConditionType = "ConfigmapReady"
)

// GetGroupVersionKind implements kmeta.OwnerRefable
func (*Function) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Function")
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (f *Function) GetConditionSet() apis.ConditionSet {
	return condSet
}

// InitializeConditions sets the initial values to the conditions.
func (fs *FunctionStatus) InitializeConditions() {
	condSet.Manage(fs).InitializeConditions()
}

// MarkServiceUnavailable updates Function status with Function Service Not Ready condition
func (fs *FunctionStatus) MarkServiceUnavailable(name string) {
	condSet.Manage(fs).MarkFalse(
		conditionServiceReady,
		"FunctionServiceUnavailable",
		"Function Service %q is not ready.", name)
}

// MarkServiceAvailable updates Function status with Function Service Is Ready condition
func (fs *FunctionStatus) MarkServiceAvailable() {
	condSet.Manage(fs).MarkTrue(conditionServiceReady)
}

// MarkSinkUnavailable updates Function status with Sink Not Ready condition
func (fs *FunctionStatus) MarkSinkUnavailable() {
	condSet.Manage(fs).MarkFalse(
		conditionSinkReady,
		"SinkUnavailable",
		"Sink is unavailable")
}

// MarkSinkAvailable updates Function status with Sink Is Ready condition
func (fs *FunctionStatus) MarkSinkAvailable() {
	condSet.Manage(fs).MarkTrue(conditionSinkReady)
}

// MarkConfigmapUnavailable updates Function status with Configmap Unavailable condition
func (fs *FunctionStatus) MarkConfigmapUnavailable(name string) {
	condSet.Manage(fs).MarkFalse(
		conditionConfigmapReady,
		"ConfigmapUnavailable",
		"Configmap is not ready")
}

// MarkConfigmapAvailable updates Function status with Configmap Is Ready condition
func (fs *FunctionStatus) MarkConfigmapAvailable() {
	condSet.Manage(fs).MarkTrue(conditionConfigmapReady)
}
