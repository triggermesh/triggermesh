/*
Copyright 2020 TriggerMesh Inc.

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

package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringerStorageAccountResourceID(t *testing.T) {
	testCases := []struct {
		name         string
		input        StorageAccountResourceID
		expectOutput string
	}{{
		name: "Valid resource ID",
		input: StorageAccountResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
			StorageAccount: "sa",
		},
		expectOutput: "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa",
	}, {
		name: "Invalid resource ID",
		input: StorageAccountResourceID{
			SubscriptionID: "",
		},
		expectOutput: "",
	}}

	for _, tc := range testCases {
		//nolint:scopelint
		t.Run(tc.name, func(t *testing.T) {
			s := tc.input.String()
			assert.Equal(t, tc.expectOutput, s)
		})
	}
}

func TestMarshalStorageAccountResourceID(t *testing.T) {
	testCases := []struct {
		name              string
		input             StorageAccountResourceID
		expectOutput      string
		expectErrContains string
	}{{
		name: "All fields are filled in",
		input: StorageAccountResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
			StorageAccount: "sa",
		},
		expectOutput: `"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa"`,
	}, {
		name: "Some fields are empty",
		input: StorageAccountResourceID{
			SubscriptionID: "s",
		},
		expectErrContains: "resource ID contains empty attributes",
	}, {
		name:              "All fields are empty",
		input:             StorageAccountResourceID{},
		expectErrContains: "resource ID contains empty attributes",
	}}

	for _, tc := range testCases {
		//nolint:scopelint
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.input)

			assert.Equal(t, tc.expectOutput, string(b))

			if errStr := tc.expectErrContains; errStr != "" {
				assert.Contains(t, err.Error(), errStr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnmarshalStorageAccountResourceID(t *testing.T) {
	testCases := []struct {
		name              string
		input             string
		expectOutput      StorageAccountResourceID
		expectErrContains string
	}{{
		name:  "Valid resource ID format",
		input: `"/subscriptions/s/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa"`,
		expectOutput: StorageAccountResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
			StorageAccount: "sa",
		},
	}, {
		name:              "Some fields are empty",
		input:             `"/subscriptions/s/resourceGroups//providers/Microsoft.Storage/storageAccounts/"`,
		expectErrContains: "resource ID contains empty attributes",
	}, {
		name:              "Invalid format",
		input:             `"/subscriptions/s"`,
		expectErrContains: "does not match expected format",
	}, {
		name:              "Invalid input",
		input:             `not_a_resource_id`,
		expectErrContains: "invalid character",
	}}

	for _, tc := range testCases {
		//nolint:scopelint
		t.Run(tc.name, func(t *testing.T) {
			resID := &StorageAccountResourceID{}
			err := json.Unmarshal([]byte(tc.input), resID)

			assert.Equal(t, tc.expectOutput, *resID)

			if errStr := tc.expectErrContains; errStr != "" {
				assert.Contains(t, err.Error(), errStr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
