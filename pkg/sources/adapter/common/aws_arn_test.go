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

package common

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
			input:       "arn:::::",
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

func TestMustParseResource(t *testing.T) {
	testCases := map[string]struct {
		input     string
		fmt       string
		expectErr bool
		expect    []string
	}{
		"input matches format, single element": {
			input:  "key/some-value",
			fmt:    "key/val",
			expect: []string{"some-value"},
		},
		"input matches format, multiple elements": {
			input:  "key1/some-value/key2/some-other-value",
			fmt:    "key1/val/key2/val",
			expect: []string{"some-value", "some-other-value"},
		},
		"only keys matter in format": {
			input:  "key1/some-value/key2/some-other-value",
			fmt:    "key1//key2/",
			expect: []string{"some-value", "some-other-value"},
		},
		"odd number of elements yields values only": {
			input:  "key1/some-value/key2/some-other-value/key3",
			fmt:    "key1/val/key2/val/key3",
			expect: []string{"some-value", "some-other-value"},
		},
		"empty input": {
			input:     "",
			fmt:       "key/val",
			expectErr: true,
		},
		"more elements than format expects": {
			input:     "key1/some-value/key2",
			fmt:       "key1/val",
			expectErr: true,
		},
		"empty format": {
			input:     "key1/some-value/key2",
			fmt:       "",
			expectErr: true,
		},
		"non-matching key": {
			input:     "some-key/some-value",
			fmt:       "key/val",
			expectErr: true,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			out, err := parseResource(tc.input, tc.fmt)

			assert.Equal(t, tc.expect, out)

			if tc.expectErr {
				assert.EqualError(t, err, newParseResourceError(tc.fmt, tc.input).Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMustParseCognitoIdentityResource(t *testing.T) {
	testCases := map[string]struct {
		input       string
		expect      string
		expectPanic bool
	}{
		"valid input": {
			input:  "identitypool/some-value",
			expect: "some-value",
		},
		"invalid input": {
			input:       "not-identitypool/some-value",
			expectPanic: true,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			var out string
			var testFn assert.PanicTestFunc = func() {
				out = MustParseCognitoIdentityResource(tc.input)
			}

			if tc.expectPanic {
				assert.PanicsWithError(t,
					newParseResourceError(expectCognitoIdentityResourceFmt, tc.input).Error(), testFn)
			} else {
				assert.NotPanics(t, testFn)
			}

			assert.Equal(t, tc.expect, out)
		})
	}
}

func TestMustParseCognitoUserPoolResource(t *testing.T) {
	testCases := map[string]struct {
		input       string
		expect      string
		expectPanic bool
	}{
		"valid input": {
			input:  "userpool/some-value",
			expect: "some-value",
		},
		"invalid input": {
			input:       "not-userpool/some-value",
			expectPanic: true,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			var out string
			var testFn assert.PanicTestFunc = func() {
				out = MustParseCognitoUserPoolResource(tc.input)
			}

			if tc.expectPanic {
				assert.PanicsWithError(t,
					newParseResourceError(expectCognitoUserPoolResourceFmt, tc.input).Error(), testFn)
			} else {
				assert.NotPanics(t, testFn)
			}

			assert.Equal(t, tc.expect, out)
		})
	}
}

func TestMustParseDynamoDBResource(t *testing.T) {
	testCases := map[string]struct {
		input       string
		expect      string
		expectPanic bool
	}{
		"valid input": {
			input:  "table/some-value",
			expect: "some-value",
		},
		"invalid input": {
			input:       "not-table/some-value",
			expectPanic: true,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			var out string
			var testFn assert.PanicTestFunc = func() {
				out = MustParseDynamoDBResource(tc.input)
			}

			if tc.expectPanic {
				assert.PanicsWithError(t,
					newParseResourceError(expectDynamoDBResourceFmt, tc.input).Error(), testFn)
			} else {
				assert.NotPanics(t, testFn)
			}

			assert.Equal(t, tc.expect, out)
		})
	}
}

func TestMustParseKinesisResource(t *testing.T) {
	testCases := map[string]struct {
		input       string
		expect      string
		expectPanic bool
	}{
		"valid input": {
			input:  "stream/some-value",
			expect: "some-value",
		},
		"invalid input": {
			input:       "not-stream/some-value",
			expectPanic: true,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			var out string
			var testFn assert.PanicTestFunc = func() {
				out = MustParseKinesisResource(tc.input)
			}

			if tc.expectPanic {
				assert.PanicsWithError(t,
					newParseResourceError(expectKinesisResourceFmt, tc.input).Error(), testFn)
			} else {
				assert.NotPanics(t, testFn)
			}

			assert.Equal(t, tc.expect, out)
		})
	}
}
