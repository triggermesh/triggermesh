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

package azureservicebustopicsource

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	servicebus "github.com/Azure/azure-service-bus-go"
	"github.com/Azure/go-autorest/autorest/to"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"
)

var sbs = &servicebus.Subscription{}

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

			event := servicebus.Message{
				ContentType:   "application/json",
				CorrelationID: "some-id",
				DeliveryCount: 1,
				SessionID:     to.StringPtr("12"),
				Data:          tc.eventData,
			}

			a := &adapter{
				logger:   loggingtesting.TestLogger(t),
				ceClient: ceClient,
				sub:      sbs,
				source:   "test-source",
			}

			err := a.handleMessage(context.Background(), &event)
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

	dataBytes, err := json.Marshal(unstructuredEvent["Data"])
	require.NoError(t, err)

	var data json.RawMessage
	err = json.Unmarshal(dataBytes, &data)
	require.NoError(t, err)

	return string(data)
}
