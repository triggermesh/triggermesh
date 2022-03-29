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
	"context"

	"knative.dev/pkg/apis"
)

// Validate makes sure that only one of the choices is properly informed.
func (v *ValueFromField) Validate(_ context.Context) *apis.FieldError {
	if v == nil {
		return nil
	}

	val := v.Value != ""
	secret := v.ValueFromSecret != nil && (v.ValueFromSecret.Name != "" || v.ValueFromSecret.Key != "")
	cm := v.ValueFromConfigMap != nil && (v.ValueFromConfigMap.Name != "" || v.ValueFromConfigMap.Key != "")

	if val && secret || val && cm || secret && cm {
		return apis.ErrMultipleOneOf("value", "valueFromSecret", "valueFromConfigMap")
	}

	if secret && (v.ValueFromSecret.Name == "" || v.ValueFromSecret.Key == "") {
		return apis.ErrMissingField("name", "key").ViaField("ValueFromSecret")
	}

	if cm && (v.ValueFromConfigMap.Name == "" || v.ValueFromConfigMap.Key == "") {
		return apis.ErrMissingField("name", "key").ViaField("ValueFromConfigMap")
	}

	return nil
}
