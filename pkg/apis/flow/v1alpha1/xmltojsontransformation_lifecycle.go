/*
Copyright 2020 TriggerMesh Inc.

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

var xmlToJSONCondSet = apis.NewLivingConditionSet(
	XMLToJSONTransformationConditionReady,
)

// GetGroupVersionKind implements kmeta.OwnerRefable
func (t *XMLToJSONTransformation) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("XMLToJSONTransformation")
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (t *XMLToJSONTransformation) GetConditionSet() apis.ConditionSet {
	return xmlToJSONCondSet
}

// InitializeConditions sets the initial values to the conditions.
func (ts *XMLToJSONTransformationStatus) InitializeConditions() {
	xmlToJSONCondSet.Manage(ts).InitializeConditions()
}

// MarkServiceUnavailable marks XMLToJSONTransformation as not ready with ServiceUnavailable reason.
func (ts *XMLToJSONTransformationStatus) MarkServiceUnavailable(name string) {
	xmlToJSONCondSet.Manage(ts).MarkFalse(
		XMLToJSONTransformationConditionReady,
		"ServiceUnavailable",
		"Service %q is not ready.", name)
}

// MarkServiceAvailable sets XMLToJSONTransformation condition to ready.
func (ts *XMLToJSONTransformationStatus) MarkServiceAvailable() {
	xmlToJSONCondSet.Manage(ts).MarkTrue(XMLToJSONTransformationConditionReady)
}
