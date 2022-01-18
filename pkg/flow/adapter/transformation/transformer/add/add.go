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
	Path  string
	Value string

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
func (a *Add) New(key, value string) transformer.Transformer {
	return &Add{
		Path:  key,
		Value: value,

		variables: a.variables,
	}
}

// Apply is a main method of Transformation that adds any type of
// variables into existing JSON.
func (a *Add) Apply(data []byte) ([]byte, error) {
	input := convert.SliceToMap(strings.Split(a.Path, "."), a.composeValue())
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

func (a *Add) retrieveVariable(key string) interface{} {
	if value := a.variables.Get(key); value != nil {
		return value
	}
	return key
}

func (a *Add) composeValue() interface{} {
	result := a.Value
	for _, key := range a.variables.ListKeys() {
		index := strings.Index(result, key)
		if index == -1 {
			continue
		}
		if result == key {
			return a.retrieveVariable(key)
		}
		result = fmt.Sprintf("%s%v%s", result[:index], a.retrieveVariable(key), result[index+len(key):])
	}
	return result
}
