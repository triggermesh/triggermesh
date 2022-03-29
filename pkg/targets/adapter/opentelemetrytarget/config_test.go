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
	"os"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/metric/sdkapi"
	"go.opentelemetry.io/otel/metric/unit"
)

const (
	tInstName          = "inst-name"
	tInstKindString    = "Histogram"
	tInstNumKindString = "Int64"
	tInstDescription   = "instrument one"
	tInstrumentsEnv    = "OPENTELEMETRY_INSTRUMENTS"
	tInstUnit          = unit.Dimensionless
)

var (
	tInstKind    = instrumentKinds[tInstKindString]
	tInstNumKind = numberKinds[tInstNumKindString]
)

var baseEnv = map[string]string{"NAMESPACE": "testns",
	"K_LOGGING_CONFIG":              "{}",
	"K_METRICS_CONFIG":              "{}",
	"OPENTELEMETRY_CORTEX_ENDPOINT": "http://test.endpoint",
}

func TestEnvironmentInstruments(t *testing.T) {

	testCases := map[string]struct {
		envInstruments      string
		expectedInstruments []sdkapi.Descriptor
		expectedErr         string
	}{
		"simple metric": {
			envInstruments: `[{"name":"inst-name","instrument":"` + tInstKindString + `","number":"` + tInstNumKindString + `"}]`,
			expectedInstruments: []sdkapi.Descriptor{
				sdkapi.NewDescriptor(tInstName, tInstKind, tInstNumKind, "", tInstUnit),
			}},
		"metric with description": {
			envInstruments: `[{"name":"` + tInstName + `","instrument":"` + tInstKindString + `","number":"` + tInstNumKindString + `","description":"` + tInstDescription + `"}]`,
			expectedInstruments: []sdkapi.Descriptor{
				sdkapi.NewDescriptor(tInstName, tInstKind, tInstNumKind, tInstDescription, tInstUnit),
			}},
		"unknown instrument": {
			envInstruments: `[{"name":"` + tInstName + `","instrument":"Unknown","number":"` + tInstNumKindString + `"}]`,
			expectedErr:    "unknown metric instrument kind",
		},
		"unknown number": {
			envInstruments: `[{"name":"` + tInstName + `","instrument":"` + tInstKindString + `","number":"Int24"}]`,
			expectedErr:    "unknown metric number kind",
		},
		"missing name": {
			envInstruments: `[{"instrument":"` + tInstKindString + `","number":"` + tInstNumKindString + `"}]`,
			expectedErr:    "metrics must include a name",
		},
		"missing instrument": {
			envInstruments: `[{"name":"` + tInstName + `","number":"` + tInstNumKindString + `"}]`,
			expectedErr:    "metrics must include an instrument kind",
		},
		"missing number": {
			envInstruments: `[{"name":"` + tInstName + `","instrument":"` + tInstKindString + `"}]`,
			expectedErr:    "metrics must include a number kind",
		},
	}

	for k, v := range baseEnv {
		t.Setenv(k, v)
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			t.Setenv(tInstrumentsEnv, tc.envInstruments)

			env := EnvAccessorCtor().(*envAccessor)
			err := envconfig.Process("", env)

			// clean up for next test case
			os.Unsetenv(tInstrumentsEnv)

			switch tc.expectedErr {
			case "":
				assert.Nil(t, err, "Envconfig processing returned an unexpected error")

			default:
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tc.expectedErr, "Expected error substring not found")
				}
				return
			}

			insts := make([]sdkapi.Descriptor, 0, len(env.Instruments))
			for _, i := range env.Instruments {
				insts = append(insts, i.Descriptor)
			}

			assert.Equal(t, tc.expectedInstruments, insts)
		})
	}
}
