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

package xmltojsontransformation

import (
	"context"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cetest "github.com/cloudevents/sdk-go/v2/client/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"knative.dev/eventing/pkg/adapter/v2"
)

const (
	tBridgeID         = "bride-abdc-0123"
	tCloudEventID     = "ce-abcd-0123"
	tCloudEventType   = "ce.test.type"
	tCloudEventSource = "ce.test.source"
	tSuccessAttribute = "success"
	tErrorAttribute   = "error"

	tXML1        = `<note><to>Tove</to></note>`
	tJSONOutput1 = "{\"note\": {\"to\": \"Tove\"}}\n"

	tFalseXML         = `"this is not xml"`
	tFalseXMLResponse = `{"Code":"request-validation","Description":"invalid XML","Details":null}`
)

func TestXSLTTransformEvents(t *testing.T) {
	testCases := map[string]struct {
		inEvent     cloudevents.Event
		expectEvent cloudevents.Event
	}{
		"transform ok": {
			inEvent: newCloudEvent(tXML1, cloudevents.ApplicationXML),

			expectEvent: newCloudEvent(tJSONOutput1, cloudevents.ApplicationJSON),
		},
		"transform error": {
			inEvent: newCloudEvent(tFalseXML, cloudevents.ApplicationXML),

			expectEvent: newCloudEvent(tFalseXMLResponse, cloudevents.ApplicationXML),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			env := &envAccessor{
				EnvConfig: adapter.EnvConfig{
					Component: tCloudEventSource,
				},
				BridgeIdentifier: tBridgeID,
			}

			ceClient, send, responses := cetest.NewMockResponderClient(t, 1)

			a := NewTarget(ctx, env, ceClient)

			go func() {
				require.NoError(t, a.Start(ctx))
			}()

			send <- tc.inEvent

			select {
			case event := <-responses:
				assert.Equal(t, tCloudEventSource, event.Event.Source())
				assert.Equal(t, string(tc.expectEvent.DataEncoded), string(event.Event.DataEncoded))

			case <-time.After(2 * time.Second):
				assert.Fail(t, "expected cloud event response was not received")
			}

		})
	}
}

type cloudEventOptions func(*cloudevents.Event)

func newCloudEvent(data, contentType string, opts ...cloudEventOptions) cloudevents.Event {
	event := cloudevents.NewEvent()

	if err := event.SetData(contentType, []byte(data)); err != nil {
		// not expected
		panic(err)
	}

	event.SetID(tCloudEventID)
	event.SetType(tCloudEventType)
	event.SetSource(tCloudEventSource)

	return event
}
