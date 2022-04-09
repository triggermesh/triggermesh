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

package ducktypes

import duckv1 "knative.dev/pkg/apis/duck/v1"

// DestinationToMap performs a conversion of duckv1.Destination to
// map[string]interface{}.
// Useful to avoid errors such as "cannot deep copy *v1.KReference" while
// setting a fields on an unstructured.Unstrucured object.
func DestinationToMap(dst *duckv1.Destination) map[string]interface{} {
	dstMap := make(map[string]interface{})

	if dst.Ref != nil {
		dstMap["ref"] = map[string]interface{}{
			"apiVersion": dst.Ref.APIVersion,
			"kind":       dst.Ref.Kind,
			"namespace":  dst.Ref.Namespace,
			"name":       dst.Ref.Name,
		}
	}
	if dst.URI != nil {
		dstMap["uri"] = dst.URI.String()
	}

	return dstMap
}
