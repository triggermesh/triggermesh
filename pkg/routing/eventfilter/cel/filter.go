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

package cel

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/cel-go/cel"
	"github.com/tidwall/gjson"

	"github.com/triggermesh/triggermesh/pkg/routing/eventfilter"
)

// ConditionalFilter structure holds both CEL Program and variable definitions
// so that it can be evaluated with the new variable values
type ConditionalFilter struct {
	Expression *cel.Program
	Variables  []Variable
}

// Variable contains the meta data required to parse event payload and execute
// CEL Program
type Variable struct {
	Name string
	Path string
	Type string
}

// Filter parses Event payload values defined as the expression variables, asserts their types,
// and executes CEL Program. If expression result is true, Event passes the filter.
func (c *ConditionalFilter) Filter(ctx context.Context, event cloudevents.Event) eventfilter.FilterResult {
	vars := make(map[string]interface{})

	for _, v := range c.Variables {
		switch v.Type {
		case "bool":
			vars[v.Name] = gjson.GetBytes(event.Data(), v.Path).Bool()
		case "int64":
			vars[v.Name] = gjson.GetBytes(event.Data(), v.Path).Int()
		case "uint64":
			vars[v.Name] = gjson.GetBytes(event.Data(), v.Path).Uint()
		case "double":
			vars[v.Name] = gjson.GetBytes(event.Data(), v.Path).Float()
		case "string":
			vars[v.Name] = gjson.GetBytes(event.Data(), v.Path).String()
		}
	}

	pass, err := eval(*c.Expression, vars)
	if err != nil || pass {
		return eventfilter.PassFilter
	}

	return eventfilter.FailFilter
}

// eval evaluates precompiled Expression with passed variables
func eval(program cel.Program, vars map[string]interface{}) (bool, error) {
	out, _, err := program.Eval(vars)
	return out.Value().(bool), err
}
