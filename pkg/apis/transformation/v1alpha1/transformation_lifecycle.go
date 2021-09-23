/*
Copyright 2020 Triggermesh Inc..

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
	TransformationConditionReady,
)

// GetGroupVersionKind implements kmeta.OwnerRefable
func (t *Transformation) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Transformation")
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (t *Transformation) GetConditionSet() apis.ConditionSet {
	return condSet
}

// InitializeConditions sets the initial values to the conditions.
func (ts *TransformationStatus) InitializeConditions() {
	condSet.Manage(ts).InitializeConditions()
}

// MarkServiceUnavailable marks Transformation as not ready with ServiceUnavailable reason.
func (ts *TransformationStatus) MarkServiceUnavailable(name string) {
	condSet.Manage(ts).MarkFalse(
		TransformationConditionReady,
		"ServiceUnavailable",
		"Service %q is not ready.", name)
}

// MarkServiceAvailable sets Transformation condition to ready.
func (ts *TransformationStatus) MarkServiceAvailable() {
	condSet.Manage(ts).MarkTrue(TransformationConditionReady)
}
