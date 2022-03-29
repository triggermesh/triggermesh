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

package opentelemetrytarget

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

type attributeValueType string

const (
	attributeValueTypeString  attributeValueType = "string"
	attributeValueTypeInt64   attributeValueType = "int"
	attributeValueTypeFloat64 attributeValueType = "float"
	attributeValueTypeBool    attributeValueType = "bool"
)

type Attribute struct {
	Key   string
	Type  attributeValueType
	Value json.RawMessage
}

func (a *Attribute) ParseAttribute() (*attribute.KeyValue, error) {
	if a.Key == "" {
		return nil, errors.New("field 'key' must be included in attributes")
	}
	if a.Type == "" {
		return nil, errors.New("field 'type' must be included in attributes")
	}
	if len(a.Value) == 0 {
		return nil, errors.New("field 'value' must be included in attributes")
	}

	switch a.Type {
	case attributeValueTypeInt64:
		var v int64
		if err := json.Unmarshal(a.Value, &v); err != nil {
			return nil, fmt.Errorf("value for %q attribute does not match type: %w", a.Key, err)
		}
		a := attribute.Int64(a.Key, v)
		return &a, nil

	case attributeValueTypeFloat64:
		var v float64
		if err := json.Unmarshal(a.Value, &v); err != nil {
			return nil, fmt.Errorf("value for %q attribute does not match type: %w", a.Key, err)
		}
		a := attribute.Float64(a.Key, v)
		return &a, nil

	case attributeValueTypeBool:
		var v bool
		if err := json.Unmarshal(a.Value, &v); err != nil {
			return nil, fmt.Errorf("value for %q attribute does not match type: %w", a.Key, err)
		}
		a := attribute.Bool(a.Key, v)
		return &a, nil

	case attributeValueTypeString:
		var v string
		if err := json.Unmarshal(a.Value, &v); err != nil {
			return nil, fmt.Errorf("value for %q attribute does not match type: %w", a.Key, err)
		}
		a := attribute.String(a.Key, v)
		return &a, nil
	}

	return nil, fmt.Errorf("unknown type %q for %q attribute", a.Type, a.Key)
}

type Measure struct {
	Name  string
	Kind  string
	Value json.RawMessage

	Attributes []Attribute
}
