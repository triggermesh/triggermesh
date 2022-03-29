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

package awslambdatarget

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustParseARN(t *testing.T) {
	testCases := map[string]struct {
		input       string
		expectPanic bool
	}{
		"valid input": {
			input:       "arn:aws:lambda:us-west-2:testproject:function:lambdadumper",
			expectPanic: false,
		},
		"invalid input": {
			input:       "arn:",
			expectPanic: true,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			var testFn assert.PanicTestFunc = func() {
				// we do not test the output because the
				// parsing logic belongs to the AWS SDK
				_ = MustParseARN(tc.input)
			}

			if tc.expectPanic {
				assert.PanicsWithValue(t, `failed to parse "`+tc.input+`": arn: not enough sections`, testFn)
			} else {
				assert.NotPanics(t, testFn)
			}
		})
	}
}
