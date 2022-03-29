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

package cloudevents

import (
	"encoding/json"

	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// OverridesJSON returns the JSON representation of a duckv1.CloudEventOverrides,
// after applying some optional transformations to it.
func OverridesJSON(ceo *duckv1.CloudEventOverrides, overrides ...ceOverrideOption) string {
	for _, o := range overrides {
		ceo = o(ceo)
	}

	var ceoStr string
	if b, err := json.Marshal(ceo); err == nil {
		ceoStr = string(b)
	}

	return ceoStr
}

// ceOverrideOption is a functional option that can alter a duckv1.CloudEventOverrides.
type ceOverrideOption func(*duckv1.CloudEventOverrides) *duckv1.CloudEventOverrides

// SetExtension returns a ceOverrideOption which sets a given CloudEvents
// extension to an arbitrary value, if this extension isn't already set.
func SetExtension(key, value string) ceOverrideOption {
	return func(ceo *duckv1.CloudEventOverrides) *duckv1.CloudEventOverrides {
		if ceo == nil {
			ceo = &duckv1.CloudEventOverrides{}
		}

		ext := &ceo.Extensions
		if *ext == nil {
			*ext = make(map[string]string, 1)
		}

		if _, isSet := (*ext)[key]; !isSet {
			(*ext)[key] = value
		}

		return ceo
	}
}
