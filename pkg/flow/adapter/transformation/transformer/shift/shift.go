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

package shift

import (
	"encoding/json"
	"strings"

	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/convert"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/storage"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/transformer"
)

var _ transformer.Transformer = (*Shift)(nil)

// Shift object implements Transformer interface.
type Shift struct {
	Path      string
	NewPath   string
	Value     string
	Separator string

	variables *storage.Storage
}

const delimeter string = ":"

// InitStep is used to figure out if this operation should
// run before main Transformations. For example, Store
// operation needs to run first to load all Pipeline variables.
var InitStep bool = false

// operationName is used to identify this transformation.
var operationName string = "shift"

// Register adds this transformation to the map which will
// be used to create Transformation pipeline.
func Register(m map[string]transformer.Transformer) {
	m[operationName] = &Shift{}
}

// SetStorage sets a shared Storage with Pipeline variables.
func (s *Shift) SetStorage(storage *storage.Storage) {
	s.variables = storage
}

// InitStep returns "true" if this Transformation should run
// as init step.
func (s *Shift) InitStep() bool {
	return InitStep
}

// New returns a new instance of Shift object.
func (s *Shift) New(key, value, separator string) transformer.Transformer {
	// doubtful scheme, review needed
	keys := strings.Split(key, delimeter)
	if len(keys) != 2 {
		return nil
	}
	return &Shift{
		Path:      keys[0],
		NewPath:   keys[1],
		Value:     value,
		Separator: separator,

		variables: s.variables,
	}
}

// Apply is a main method of Transformation that moves existing
// values to a new locations.
func (s *Shift) Apply(eventID string, data []byte) ([]byte, error) {
	oldPath := convert.SliceToMap(strings.Split(s.Path, s.Separator), "")

	var event interface{}
	if err := json.Unmarshal(data, &event); err != nil {
		return data, err
	}

	newEvent, value := extractValue(event, oldPath)
	if s.Value != "" {
		if !equal(s.retrieveInterface(eventID, s.Value), value) {
			return data, nil
		}
	}
	if value == nil {
		return data, nil
	}

	newPath := convert.SliceToMap(strings.Split(s.NewPath, s.Separator), value)
	result := convert.MergeJSONWithMap(newEvent, newPath)
	output, err := json.Marshal(result)
	if err != nil {
		return data, err
	}

	return output, nil
}

func (s *Shift) retrieveInterface(eventID, key string) interface{} {
	if value := s.variables.Get(eventID, key); value != nil {
		return value
	}
	return key
}

func extractValue(source interface{}, path map[string]interface{}) (map[string]interface{}, interface{}) {
	var ok bool
	var result interface{}
	sourceMap := make(map[string]interface{})
	for k, v := range path {
		switch value := v.(type) {
		case float64, bool, string:
			sourceMap, ok = source.(map[string]interface{})
			if !ok {
				break
			}
			result = sourceMap[k]
			delete(sourceMap, k)
		case []interface{}:
			if k != "" {
				// array is inside the object
				// {"foo":[{},{},{}]}
				sourceMap, ok = source.(map[string]interface{})
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

			m, ok := value[index].(map[string]interface{})
			if ok {
				sourceArr[index], result = extractValue(sourceArr[index].(map[string]interface{}), m)
				sourceMap[k] = sourceArr
				break
			}
			result = sourceArr[index]
			sourceMap[k] = sourceArr[:index]
			if len(sourceArr) > index {
				sourceMap[k] = append(sourceArr[:index], sourceArr[index+1:]...)
			}
		case map[string]interface{}:
			if k == "" {
				result = source
				break
			}
			sourceMap, ok = source.(map[string]interface{})
			if !ok {
				break
			}
			if _, ok := sourceMap[k]; !ok {
				break
			}
			sourceMap[k], result = extractValue(sourceMap[k], value)
		case nil:
			sourceMap[k] = nil
		}
	}
	return sourceMap, result
}

func equal(a, b interface{}) bool {
	switch value := b.(type) {
	case string:
		v, ok := a.(string)
		if ok && v == value {
			return true
		}
	case bool:
		v, ok := a.(bool)
		if ok && v == value {
			return true
		}
	case float64:
		v, ok := a.(float64)
		if ok && v == value {
			return true
		}
	}
	return false
}
