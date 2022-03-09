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

package convert

import (
	"strconv"
	"strings"
)

// SliceToMap converts string slice into map that can be encoded into JSON.
func SliceToMap(path []string, value interface{}) map[string]interface{} {
	var array bool
	var index int
	i := strings.Index(path[0], "[")
	if i > -1 && len(path[0]) > i+1 {
		indexStr := path[0][i+1 : len(path[0])-1]
		indexInt, err := strconv.Atoi(indexStr)
		if err == nil {
			index = indexInt
			array = true
			path[0] = path[0][:i]
		}
	}

	if len(path) == 1 {
		if !array {
			return map[string]interface{}{
				path[0]: value,
			}
		}
		arr := make([]interface{}, index+1)
		arr[index] = value
		return map[string]interface{}{
			path[0]: arr,
		}
	}

	key := path[0]
	path = path[1:]
	m := SliceToMap(path, value)
	if !array {
		return map[string]interface{}{
			key: m,
		}
	}
	arr := make([]interface{}, index+1)
	arr[index] = m
	return map[string]interface{}{
		key: arr,
	}
}

// MergeJSONWithMap accepts interface (effectively, JSON) and a map and merges them together.
// Source map keys are being overwritten by appendix keys if they overlap.
func MergeJSONWithMap(source, appendix interface{}) interface{} {
	switch appendixValue := appendix.(type) {
	case nil:
		return source
	case float64, bool, string:
		return appendixValue
	case []interface{}:
		sourceInterface, ok := source.([]interface{})
		if !ok {
			return appendixValue
		}
		resArrLen := len(sourceInterface)
		if len(appendixValue) > resArrLen {
			resArrLen = len(appendixValue)
		}
		resArr := make([]interface{}, resArrLen)
		for i := range resArr {
			var a, b interface{}
			if i < len(appendixValue) {
				b = appendixValue[i]
			}
			if i < len(sourceInterface) {
				a = sourceInterface[i]
			}
			resArr[i] = MergeJSONWithMap(a, b)
		}
		source = resArr
	case map[string]interface{}:
		switch s := source.(type) {
		case float64, bool, string:
			return appendixValue
		case nil:
			source = make(map[string]interface{})
			return MergeJSONWithMap(source, appendixValue)
		case map[string]interface{}:
			for k, v := range appendixValue {
				if k == "" {
					return MergeJSONWithMap(s, v)
				}
				s[k] = MergeJSONWithMap(s[k], v)
			}
			source = s
		case []interface{}:
			for k, v := range appendixValue {
				if k == "" {
					return MergeJSONWithMap(s, v)
				}
			}
		}
	}
	return source
}
