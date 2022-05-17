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

package azureservicebussource

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
)

func TestHandleMessage(t *testing.T) {
	testCases := []struct {
		name            string
		eventData       []byte
		expectEventData interface{}
	}{
		{
			name:            "Data is raw bytes",
			eventData:       []byte{'t', 'e', 's', 't'},
			expectEventData: `"dGVzdA=="`, // base64-encoded "test"
		},
		{
			name:            "Data is a JSON object",
			eventData:       []byte(`{"test": null}`),
			expectEventData: `{"test":null}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ceClient := adaptertest.NewTestClient()

			msg := &Message{
				ReceivedMessage: &azservicebus.ReceivedMessage{
					Body: tc.eventData,
				},
			}

			a := &adapter{
				ceClient: ceClient,
				msgPrcsr: &defaultMessageProcessor{},
			}

			err := a.handleMessage(context.Background(), msg)
			assert.NoError(t, err)

			events := ceClient.Sent()
			require.Len(t, events, 1)

			// ensure the sent event has the expected encoding (base64 / raw JSON)
			eventDataStr := extractDataFromEvent(t, events[0].Data())
			assert.Equal(t, tc.expectEventData, eventDataStr)
		})
	}
}

func extractDataFromEvent(t *testing.T, b []byte) string {
	unstructuredEvent := make(map[string]interface{})
	err := json.Unmarshal(b, &unstructuredEvent)
	require.NoError(t, err)

	dataBytes, err := json.Marshal(unstructuredEvent["Body"])
	require.NoError(t, err)

	var data json.RawMessage
	err = json.Unmarshal(dataBytes, &data)
	require.NoError(t, err)

	return string(data)
}

func TestParseServiceBusResourceID(t *testing.T) {
	const resourceIDPrefix = "/subscriptions/s/resourceGroups/rg/providers"

	testCases := []struct {
		name         string
		input        string
		expectErr    bool
		expectNs     string
		expectRes    string
		expectSubRes string
	}{
		{
			name:         "Valid Queue ID",
			input:        resourceIDPrefix + "/Microsoft.ServiceBus/namespaces/ns/queues/q",
			expectErr:    false,
			expectNs:     "ns",
			expectRes:    "q",
			expectSubRes: "",
		},
		{
			name:         "Valid Topic subscription ID",
			input:        resourceIDPrefix + "/Microsoft.ServiceBus/namespaces/ns/topics/t/subscriptions/s",
			expectErr:    false,
			expectNs:     "ns",
			expectRes:    "t",
			expectSubRes: "s",
		},
		{
			name:      "Not the Service Bus provider",
			input:     resourceIDPrefix + "/Microsoft.EventHubs/namespaces/ns/queues/q",
			expectErr: true,
		},
		{
			name:      "Not a supported Service Bus entity",
			input:     resourceIDPrefix + "/Microsoft.ServiceBus/namespaces/ns/notsupported/x",
			expectErr: true,
		},
		{
			name:      "Queue ID with a sub-resource",
			input:     resourceIDPrefix + "/Microsoft.ServiceBus/namespaces/ns/queues/q/subscription/s",
			expectErr: true,
		},
		{
			name:      "Topic ID without sub-resource",
			input:     resourceIDPrefix + "/Microsoft.ServiceBus/namespaces/ns/topics/t",
			expectErr: true,
		},
		{
			name:      "Topic ID with a sub-resource that is not a subscription",
			input:     resourceIDPrefix + "/Microsoft.ServiceBus/namespaces/ns/topics/t/notsupported/x",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := parseServiceBusResourceID(tc.input)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, out)
				return
			}

			assert.NoError(t, err)
			require.NotNil(t, out)

			assert.Equal(t, tc.expectNs, out.Namespace, "Unexpected resource namespace")
			assert.Equal(t, tc.expectRes, out.ResourceName, "Unexpected resource name")
			assert.Equal(t, tc.expectSubRes, out.SubResourceName, "Unexpected sub-resource name")
		})
	}
}
