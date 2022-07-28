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

package common

// ReadValue returns the source object item located at the requested path.
func ReadValue(source interface{}, path map[string]interface{}) interface{} {
	var result interface{}
	for k, v := range path {
		switch value := v.(type) {
		case float64, bool, string:
			sourceMap, ok := source.(map[string]interface{})
			if !ok {
				break
			}
			result = sourceMap[k]
		case []interface{}:
			if k != "" {
				// array is inside the object
				// {"foo":[{},{},{}]}
				sourceMap, ok := source.(map[string]interface{})
				if !ok {
					break
				}
				source, ok = sourceMap[k]
				if !ok {
					break
				}
			}
			// array is a root object
			// [{},{},{}]
			sourceArr, ok := source.([]interface{})
			if !ok {
				break
			}

			index := len(value) - 1
			if index >= len(sourceArr) {
				break
			}
			result = ReadValue(sourceArr[index], value[index].(map[string]interface{}))
		case map[string]interface{}:
			if k == "" {
				result = source
				break
			}
			sourceMap, ok := source.(map[string]interface{})
			if !ok {
				break
			}
			if _, ok := sourceMap[k]; !ok {
				break
			}
			result = ReadValue(sourceMap[k], value)
		}
	}
	return result
}
