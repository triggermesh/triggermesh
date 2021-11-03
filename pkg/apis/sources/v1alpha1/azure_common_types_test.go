/*
Copyright 2021 TriggerMesh Inc.

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

func TestStringerAzureResourceID(t *testing.T) {
	testCases := []struct {
		name         string
		input        AzureResourceID
		expectOutput string
	}{{
		name: "Valid resource ID (subscription)",
		input: AzureResourceID{
			SubscriptionID: "s",
		},
		expectOutput: "/subscriptions/s",
	}, {
		name: "Valid resource ID (resource group)",
		input: AzureResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
		},
		expectOutput: "/subscriptions/s/resourceGroups/rg",
	}, {
		name: "Valid resource ID (resource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			ResourceType:     "rt",
			ResourceName:     "rn",
		},
		expectOutput: "/subscriptions/s/resourceGroups/rg/providers/rp/rt/rn",
	}, {
		name: "Valid resource ID (namespace)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			ResourceType:     "namespaces",
			ResourceName:     "ns",
		},
		expectOutput: "/subscriptions/s/resourceGroups/rg/providers/rp/namespaces/ns",
	}, {
		name: "Valid resource ID (namespaced resource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			Namespace:        "ns",
			ResourceType:     "rt",
			ResourceName:     "rn",
		},
		expectOutput: "/subscriptions/s/resourceGroups/rg/providers/rp/namespaces/ns/rt/rn",
	}, {
		name: "Valid resource ID (namespaced subresource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			Namespace:        "ns",
			ResourceType:     "rt",
			ResourceName:     "rn",
			SubResourceType:  "srt",
			SubResourceName:  "srn",
		},
		expectOutput: "/subscriptions/s/resourceGroups/rg/providers/rp/namespaces/ns/rt/rn/srt/srn",
	}, {
		name: "Invalid resource ID (subscription)",
		input: AzureResourceID{
			SubscriptionID: "",
		},
		expectOutput: "",
	}, {
		name: "Invalid resource ID (resource group)",
		input: AzureResourceID{
			SubscriptionID: "",
			ResourceGroup:  "rg",
		},
		expectOutput: "",
	}, {
		name: "Invalid resource ID (resource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			ResourceType:     "",
			ResourceName:     "rn",
		},
		expectOutput: "",
	}, {
		name: "Invalid resource ID (namespaced resource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			Namespace:        "ns",
			ResourceType:     "",
			ResourceName:     "rn",
		},
		expectOutput: "",
	}, {
		name: "Invalid resource ID (namespaced subresource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			Namespace:        "ns",
			ResourceType:     "rt",
			ResourceName:     "rn",
			SubResourceType:  "",
			SubResourceName:  "srn",
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

func TestMarshalAzureResourceID(t *testing.T) {
	testCases := []struct {
		name              string
		input             AzureResourceID
		expectOutput      string
		expectErrContains string
	}{{
		name: "All fields are filled in (resource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			ResourceType:     "rt",
			ResourceName:     "rn",
		},
		expectOutput: `"/subscriptions/s/resourceGroups/rg/providers/rp/rt/rn"`,
	}, {
		name: "All fields are filled in (namespace)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			ResourceType:     "namespaces",
			ResourceName:     "ns",
		},
		expectOutput: `"/subscriptions/s/resourceGroups/rg/providers/rp/namespaces/ns"`,
	}, {
		name: "All fields are filled in (namespaced resource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			Namespace:        "ns",
			ResourceType:     "rt",
			ResourceName:     "rn",
		},
		expectOutput: `"/subscriptions/s/resourceGroups/rg/providers/rp/namespaces/ns/rt/rn"`,
	}, {
		name: "All fields are filled in (namespaced subresource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			Namespace:        "ns",
			ResourceType:     "rt",
			ResourceName:     "rn",
			SubResourceType:  "srt",
			SubResourceName:  "srn",
		},
		expectOutput: `"/subscriptions/s/resourceGroups/rg/providers/rp/namespaces/ns/rt/rn/srt/srn"`,
	}, {
		name: "Resource fields are empty",
		input: AzureResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
		},
		expectOutput: `"/subscriptions/s/resourceGroups/rg"`,
	}, {
		name: "Some required fields are empty (resource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			ResourceType:     "",
			ResourceName:     "rn",
		},
		expectErrContains: "resource ID contains empty attributes",
	}, {
		name: "Some required fields are empty (namespaced resource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			Namespace:        "ns",
			ResourceType:     "",
			ResourceName:     "rn",
		},
		expectErrContains: "resource ID contains empty attributes",
	}, {
		name: "Some required fields are empty (namespaced subresource)",
		input: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			Namespace:        "ns",
			ResourceType:     "rt",
			ResourceName:     "rn",
			SubResourceType:  "",
			SubResourceName:  "srn",
		},
		expectErrContains: "resource ID contains empty attributes",
	}, {
		name:              "All fields are empty",
		input:             AzureResourceID{},
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

func TestUnmarshalAzureResourceID(t *testing.T) {
	testCases := []struct {
		name              string
		input             string
		expectOutput      AzureResourceID
		expectErrContains string
	}{{
		name:  "Valid resource ID format (resource)",
		input: `"/subscriptions/s/resourceGroups/rg/providers/rp/rt/rn"`,
		expectOutput: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "rp",
			ResourceType:     "rt",
			ResourceName:     "rn",
		},
	}, {
		name:  "Valid resource ID format (resource group)",
		input: `"/subscriptions/s/resourceGroups/rg"`,
		expectOutput: AzureResourceID{
			SubscriptionID:   "s",
			ResourceGroup:    "rg",
			ResourceProvider: "",
			ResourceType:     "",
			ResourceName:     "",
		},
	}, {
		name:              "Some required fields are empty",
		input:             `"/subscriptions/s/resourceGroups/rg/providers/rp//rn"`,
		expectErrContains: "resource ID contains empty attributes",
	}, {
		name:              "Invalid format",
		input:             `"/subscriptions/s/resourceGroups/rg/"`,
		expectErrContains: "does not match expected format",
	}, {
		name:              "Invalid input",
		input:             `not_a_resource_id`,
		expectErrContains: "invalid character",
	}}

	for _, tc := range testCases {
		//nolint:scopelint
		t.Run(tc.name, func(t *testing.T) {
			resID := &AzureResourceID{}
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

func TestStringerEventHubResourceID(t *testing.T) {
	testCases := []struct {
		name         string
		input        EventHubResourceID
		expectOutput string
	}{{
		name: "Valid Event Hubs instance resource ID",
		input: EventHubResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
			Namespace:      "ns",
			EventHub:       "eh",
		},
		expectOutput: "/subscriptions/s/resourceGroups/rg/providers/Microsoft.EventHub/namespaces/ns/eventHubs/eh",
	}, {
		name: "Valid Event Hubs namespace resource ID",
		input: EventHubResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
			Namespace:      "ns",
		},
		expectOutput: "/subscriptions/s/resourceGroups/rg/providers/Microsoft.EventHub/namespaces/ns",
	}, {
		name: "Invalid resource ID",
		input: EventHubResourceID{
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

func TestMarshalEventHubResourceID(t *testing.T) {
	testCases := []struct {
		name              string
		input             EventHubResourceID
		expectOutput      string
		expectErrContains string
	}{{
		name: "All fields are filled in",
		input: EventHubResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
			Namespace:      "ns",
			EventHub:       "eh",
		},
		expectOutput: `"/subscriptions/s/resourceGroups/rg/providers/Microsoft.EventHub/namespaces/ns/eventHubs/eh"`,
	}, {
		name: "Only EventHub field is empty",
		input: EventHubResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
			Namespace:      "ns",
		},
		expectOutput: `"/subscriptions/s/resourceGroups/rg/providers/Microsoft.EventHub/namespaces/ns"`,
	}, {
		name: "Some required fields are empty",
		input: EventHubResourceID{
			SubscriptionID: "s",
		},
		expectErrContains: "resource ID contains empty attributes",
	}, {
		name:              "All fields are empty",
		input:             EventHubResourceID{},
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

func TestUnmarshalEventHubResourceID(t *testing.T) {
	testCases := []struct {
		name              string
		input             string
		expectOutput      EventHubResourceID
		expectErrContains string
	}{{
		name:  "Valid Event Hubs instance resource ID format",
		input: `"/subscriptions/s/resourceGroups/rg/providers/Microsoft.EventHub/namespaces/ns/eventHubs/eh"`,
		expectOutput: EventHubResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
			Namespace:      "ns",
			EventHub:       "eh",
		},
	}, {
		name:  "Valid Event Hubs namespace resource ID format",
		input: `"/subscriptions/s/resourceGroups/rg/providers/Microsoft.EventHub/namespaces/ns"`,
		expectOutput: EventHubResourceID{
			SubscriptionID: "s",
			ResourceGroup:  "rg",
			Namespace:      "ns",
			EventHub:       "",
		},
	}, {
		name:              "Some required fields are empty",
		input:             `"/subscriptions/s/resourceGroups//providers/Microsoft.EventHub/namespaces//eventHubs/"`,
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
			resID := &EventHubResourceID{}
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
