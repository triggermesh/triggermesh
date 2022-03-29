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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestMeasuresAttributesParsing(t *testing.T) {
	testCases := map[string]struct {
		in                 string
		expectedAttributes []*attribute.KeyValue
		expectedErr        string
	}{
		"string attributes": {
			in: `
			{
				"name":"total_requests",
				"kind":"Counter",
				"value":1,
				"attributes":[
					{"key":"host","value":"tm1","type":"string"},
					{"key":"type","value":"large","type":"string"}
				]
			}
			`,
			expectedAttributes: []*attribute.KeyValue{
				createStringAttribute("host", "tm1"),
				createStringAttribute("type", "large"),
			},
		},

		"bool attribute": {
			in: `
			{
				"name":"sessions",
				"kind":"UpDown",
				"value":13,
				"attributes":[
					{"key":"is_admin","value":true,"type":"bool"}
				]
			}
			`,
			expectedAttributes: []*attribute.KeyValue{
				createBoolAttribute("is_admin", true),
			},
		},

		"float attribute": {
			in: `
			{
				"name":"watermelon_count",
				"kind":"UpDownCounter",
				"value":35,
				"attributes":[
					{"key":"weight_range_min","value":0.5,"type":"float"},
					{"key":"weight_range_max","value":1.5,"type":"float"}
				]
			}
			`,
			expectedAttributes: []*attribute.KeyValue{
				createFloatAttribute("weight_range_min", 0.5),
				createFloatAttribute("weight_range_max", 1.5),
			},
		},

		"int attribute": {
			in: `
			{
				"name":"covid_cases",
				"value":0,
				"attributes":[
					{"key":"year","value":2024,"type":"int"}
				]
			}
			`,
			expectedAttributes: []*attribute.KeyValue{
				createIntAttribute("year", 2024),
			},
		},

		"key not informed": {
			in: `
			{
				"name":"foo",
				"value":"bar",
				"attributes":[
					{"value":1,"type":"int"}
				]
			}
			`,
			expectedErr: "field 'key' must be included in attributes",
		},

		"value not informed": {
			in: `
			{
				"name":"foo",
				"value":"bar",
				"attributes":[
					{"key":"year","type":"int"}
				]
			}
			`,
			expectedErr: "field 'value' must be included in attributes",
		},

		"type not informed": {
			in: `
			{
				"name":"foo",
				"value":"bar",
				"attributes":[
					{"key":"year","value":1}
				]
			}
			`,
			expectedErr: "field 'type' must be included in attributes",
		},

		"wrong string attribute": {
			in: `
			{
				"name":"total_requests",
				"kind":"Counter",
				"value":1,
				"attributes":[
					{"key":"host","value":12,"type":"string"}
				]
			}
			`,
			expectedErr: "attribute does not match type",
		},

		"wrong bool attribute": {
			in: `
			{
				"name":"sessions",
				"kind":"UpDownCounter",
				"value":13,
				"attributes":[
					{"key":"is_admin","value":"true","type":"bool"}
				]
			}
			`,
			expectedErr: "attribute does not match type",
		},

		"wrong float attribute": {
			in: `
			{
				"name":"watermelon_count",
				"kind":"UpDownCounter",
				"value":35,
				"attributes":[
					{"key":"weight_range_min","value":true,"type":"float"}
				]
			}
			`,
			expectedErr: "attribute does not match type",
		},

		"wrong int attribute": {
			in: `
			{
				"name":"covid_cases",
				"value":0,
				"attributes":[
					{"key":"year","value":1.3,"type":"int"}
				]
			}
			`,
			expectedErr: "attribute does not match type",
		},

		"unknown attribute type": {
			in: `
			{
				"name":"covid_cases",
				"value":0,
				"attributes":[
					{"key":"year","value":1.3,"type":"foobar"}
				]
			}
			`,
			expectedErr: "unknown type",
		},
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			measure := Measure{}
			err := json.Unmarshal([]byte(tc.in), &measure)
			require.NoError(t, err)

			attrs := []*attribute.KeyValue{}
			for _, ma := range measure.Attributes {
				attr, err := ma.ParseAttribute()

				switch tc.expectedErr {
				case "":
					require.Nil(t, err, "Attribute parser returned an unexpected error")

				default:
					require.Error(t, err)
					require.Contains(t, err.Error(), tc.expectedErr, "Expected error substring not found")
					return
				}
				attrs = append(attrs, attr)
			}

			assert.Equal(t, tc.expectedAttributes, attrs)
		})
	}
}

func createStringAttribute(key string, value string) *attribute.KeyValue {
	return &attribute.KeyValue{
		Key:   attribute.Key(key),
		Value: attribute.StringValue(value),
	}
}

func createBoolAttribute(key string, value bool) *attribute.KeyValue {
	return &attribute.KeyValue{
		Key:   attribute.Key(key),
		Value: attribute.BoolValue(value),
	}
}

func createFloatAttribute(key string, value float64) *attribute.KeyValue {
	return &attribute.KeyValue{
		Key:   attribute.Key(key),
		Value: attribute.Float64Value(value),
	}
}

func createIntAttribute(key string, value int64) *attribute.KeyValue {
	return &attribute.KeyValue{
		Key:   attribute.Key(key),
		Value: attribute.Int64Value(value),
	}
}
