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

package add

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/convert"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/storage"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/transformer"
)

var _ transformer.Transformer = (*Add)(nil)

// Add object implements Transformer interface.
type Add struct {
	Path      string
	Value     string
	Separator string

	variables *storage.Storage
}

// InitStep is used to figure out if this operation should
// run before main Transformations. For example, Store
// operation needs to run first to load all Pipeline variables.
var InitStep bool = false

// operationName is used to identify this transformation.
var operationName string = "add"

// Register adds this transformation to the map which will
// be used to create Transformation pipeline.
func Register(m map[string]transformer.Transformer) {
	m[operationName] = &Add{}
}

// SetStorage sets a shared Storage with Pipeline variables.
func (a *Add) SetStorage(storage *storage.Storage) {
	a.variables = storage
}

// InitStep returns "true" if this Transformation should run
// as init step.
func (a *Add) InitStep() bool {
	return InitStep
}

// New returns a new instance of Add object.
func (a *Add) New(key, value, separator string) transformer.Transformer {
	return &Add{
		Path:      key,
		Value:     value,
		Separator: separator,

		variables: a.variables,
	}
}

// Apply is a main method of Transformation that adds any type of
// variables into existing JSON.
func (a *Add) Apply(eventID string, data []byte) ([]byte, error) {
	input := convert.SliceToMap(strings.Split(a.Path, a.Separator), a.composeValue(eventID))
	var event interface{}
	if err := json.Unmarshal(data, &event); err != nil {
		return data, err
	}

	result := convert.MergeJSONWithMap(event, input)
	output, err := json.Marshal(result)
	if err != nil {
		return data, err
	}

	return output, nil
}

func (a *Add) retrieveVariable(eventID, key string) interface{} {
	if value := a.variables.Get(eventID, key); value != nil {
		return value
	}
	return key
}

func (a *Add) composeValue(eventID string) interface{} {
	result := a.Value
	for _, key := range a.variables.ListEventVariables(eventID) {
		// limit the number of iterations to prevent the loop if
		// "add" variable is not updating the result (variable is not defined).
		variableKeysInResult := strings.Count(result, key)
		for i := 0; i <= variableKeysInResult; i++ {
			keyIndex := strings.Index(result, key)
			if keyIndex == -1 {
				continue
			}

			storedValue := a.retrieveVariable(eventID, key)

			if result == key {
				return storedValue
			}

			openingBracketIndex := -1
			closingBracketIndex := -1
			for i := keyIndex; i >= 0; i-- {
				if string(result[i]) == "(" {
					openingBracketIndex = i
					break
				}
			}
			for i := keyIndex; i < len(result); i++ {
				if string(result[i]) == ")" {
					closingBracketIndex = i
					break
				}
			}

			// there is no brackets in the value
			if (openingBracketIndex == -1 || closingBracketIndex == -1) ||
				// brackets are screened with "\" symbol
				((openingBracketIndex > 0 && string(result[openingBracketIndex-1]) == "\\") ||
					string(result[closingBracketIndex-1]) == "\\") ||
				// brackets are not surrounding the key
				!(openingBracketIndex < keyIndex && closingBracketIndex >= keyIndex+len(key)) {
				result = fmt.Sprintf("%s%v%s", result[:keyIndex], storedValue, result[keyIndex+len(key):])
				continue
			}

			if storedValue == key {
				// stored value that equals the variable key means no stored value is available
				result = fmt.Sprintf("%s%s", result[:openingBracketIndex], result[closingBracketIndex+1:])
				continue
			}

			result = result[:openingBracketIndex] + result[openingBracketIndex+1:]
			result = result[:closingBracketIndex-1] + result[closingBracketIndex:]
			result = fmt.Sprintf("%s%v%s", result[:keyIndex-1], storedValue, result[keyIndex+len(key)-1:])
		}
	}
	return result
}
