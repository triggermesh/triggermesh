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

package apis

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/stretchr/testify/assert"
)

func TestStringARN(t *testing.T) {
	in := ARN{
		Partition: "aws",
		Service:   "some-service",
		Region:    "us-test-0",
		AccountID: "123456789012",
		Resource:  "some-type/some-id",
	}

	expect, got := arn.ARN(in).String(), in.String()
	assert.Equal(t, expect, got)
}

func TestMarshalARN(t *testing.T) {
	testCases := []struct {
		name         string
		input        ARN
		expectOutput string
	}{{
		name: "All fields are filled in",
		input: ARN{
			Partition: "aws",
			Service:   "some-service",
			Region:    "us-test-0",
			AccountID: "123456789012",
			Resource:  "some-type/some-id",
		},
		expectOutput: `"arn:aws:some-service:us-test-0:123456789012:some-type/some-id"`,
	}, {
		name: "Some fields are empty",
		input: ARN{
			Region: "us-test-0",
		},
		expectOutput: `"arn:::us-test-0::"`,
	}}

	for _, tc := range testCases {
		//nolint:scopelint
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectOutput, string(b))
		})
	}
}

func TestUnmarshalARN(t *testing.T) {
	testCases := []struct {
		name              string
		input             string
		expectOutput      ARN
		expectErrContains string
	}{{
		name:  "Valid ARN format",
		input: `"arn:partition:service:region:account-id:resource-type/resource-id"`,
		expectOutput: ARN{
			Partition: "partition",
			Service:   "service",
			Region:    "region",
			AccountID: "account-id",
			Resource:  "resource-type/resource-id",
		},
	}, {
		name:  "Some fields are empty",
		input: `"arn:::region::"`,
		expectOutput: ARN{
			Region: "region",
		},
	}, {
		name:              "Invalid number of sections",
		input:             `"arn:partition:"`,
		expectErrContains: "not enough sections",
	}, {
		name:              "Invalid format",
		input:             `"not-arn:"`,
		expectErrContains: "invalid prefix",
	}, {
		name:              "Invalid input",
		input:             `not_a_resource_id`,
		expectErrContains: "invalid character",
	}}

	for _, tc := range testCases {
		//nolint:scopelint
		t.Run(tc.name, func(t *testing.T) {
			arn := &ARN{}
			err := json.Unmarshal([]byte(tc.input), arn)

			assert.Equal(t, tc.expectOutput, *arn)

			if errStr := tc.expectErrContains; errStr != "" {
				assert.Contains(t, err.Error(), errStr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
