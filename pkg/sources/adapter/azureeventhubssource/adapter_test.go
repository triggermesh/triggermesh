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

package azureeventhubssource

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"
)

func TestHandleMessage(t *testing.T) {
	const ceSource = "fake.source"
	const ceType = "fake.type"

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

			event := &azeventhubs.ReceivedEventData{
				EventData: azeventhubs.EventData{
					Body: tc.eventData,
				},
			}

			a := &adapter{
				runtimeInfo: &azeventhubs.EventHubProperties{
					Name: "testHub",
				},
				logger:   loggingtesting.TestLogger(t),
				ceClient: ceClient,
				msgPrcsr: &defaultMessageProcessor{
					ceSource: ceSource,
					ceType:   ceType,
				},
			}

			err := a.handleMessage(context.Background(), event)
			assert.NoError(t, err)

			events := ceClient.Sent()
			require.Len(t, events, 1)

			// ensure the sent event has the expected encoding (base64 / raw JSON)
			eventDataStr := extractDataFromEvent(t, events[0].Data())
			assert.Equal(t, tc.expectEventData, eventDataStr)

			assert.Equal(t, ceSource, events[0].Source())
			assert.Equal(t, ceType, events[0].Type())
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
