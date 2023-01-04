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

package parse

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/convert"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/storage"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/transformer"
)

var _ transformer.Transformer = (*Parse)(nil)

// Parse object implements Transformer interface.
type Parse struct {
	Path      string
	Value     string
	Separator string

	variables *storage.Storage
}

// InitStep is used to figure out if this operation should
// run before main Transformations. For example, Store
// operation needs to run first to load all Pipeline variables.
var InitStep bool = true

// operationName is used to identify this transformation.
var operationName string = "parse"

// Register adds this transformation to the map which will
// be used to create Transformation pipeline.
func Register(m map[string]transformer.Transformer) {
	m[operationName] = &Parse{}
}

// SetStorage sets a shared Storage with Pipeline variables.
func (p *Parse) SetStorage(storage *storage.Storage) {
	p.variables = storage
}

// InitStep returns "true" if this Transformation should run
// as init step.
func (p *Parse) InitStep() bool {
	return InitStep
}

// New returns a new instance of Parse object.
func (p *Parse) New(key, value, separator string) transformer.Transformer {
	return &Parse{
		Path:      key,
		Value:     value,
		Separator: separator,

		variables: p.variables,
	}
}

// Apply is a main method of Transformation that parse JSON values
// into variables that can be used by other Transformations in a pipeline.
func (p *Parse) Apply(eventID string, data []byte) ([]byte, error) {
	path := convert.SliceToMap(strings.Split(p.Path, p.Separator), "")

	switch p.Value {
	case "json", "JSON":
		var event interface{}
		if err := json.Unmarshal(data, &event); err != nil {
			return data, err
		}
		jsonValue, err := parseJSON(common.ReadValue(event, path))
		if err != nil {
			return data, err
		}
		newObject := convert.SliceToMap(strings.Split(p.Path, p.Separator), jsonValue)
		return json.Marshal(convert.MergeJSONWithMap(event, newObject))
	default:
		return data, fmt.Errorf("parse operation does not support %q type of value", p.Value)
	}
}

func parseJSON(data interface{}) (interface{}, error) {
	str, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast the value to string type")
	}
	var object interface{}
	if err := json.Unmarshal([]byte(str), &object); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}
	return object, nil
}
