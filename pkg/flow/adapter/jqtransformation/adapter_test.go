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

package jqtransformation

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cetest "github.com/cloudevents/sdk-go/v2/client/test"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"knative.dev/eventing/pkg/adapter/v2"
)

const (
	tCloudEventID     = "ce-abcd-0123"
	tCloudEventType   = "ce.test.type"
	tCloudEventSource = "ce.test.source"

	tJSONQuery   = ".greeting | .."
	tJSON1       = `{"name":{"first":"bob","last":"smith"},"age": 42,"greeting": "hello"}`
	tJSONOutput1 = "\"hello\""

	tJSONQuery2  = ".[1].greeting"
	tJSON2       = `[{"name":{"first":"bob","last":"smith"},"age": 42},{"greeting": "hello"}]`
	tJSONOutput2 = "\"hello\""

	tInvalidJSON         = `"I am not valid JSON"`
	tInvalidJSONResponse = `{"Code":"request-parsing","Description":"expected an object but got: string (\"I am not valid JSON\")","Details":null}`
)

func TestSink(t *testing.T) {
	testCases := map[string]struct {
		inEvent     cloudevents.Event
		expectEvent cloudevents.Event
	}{
		"sink ok": {
			inEvent:     newCloudEvent(t, tJSON1, cloudevents.ApplicationJSON),
			expectEvent: newCloudEvent(t, tJSONOutput1, cloudevents.ApplicationJSON),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := ioutil.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Equal(t, tJSON1, string(body))
				fmt.Fprintf(w, "OK")
			}))
			defer svr.Close()

			env := &envAccessor{
				EnvConfig: adapter.EnvConfig{
					Component: tCloudEventSource,
					Sink:      svr.URL,
				},
				Query: tJSONQuery,
			}
			ctx := context.Background()
			c, err := cloudevents.NewClientHTTP()
			assert.NoError(t, err)
			a := NewAdapter(ctx, env, c)

			go func() {
				if err := a.Start(ctx); err != nil {
					assert.FailNow(t, "could not start test adapter")
				}
			}()

			response := sendCE(t, &tc.inEvent, svr.URL)
			assert.NotEqual(t, cloudevents.IsUndelivered(response), response)
		})
	}
}

func TestReplier(t *testing.T) {
	testCases := map[string]struct {
		query       string
		inEvent     cloudevents.Event
		expectEvent cloudevents.Event
	}{
		"transform object ok": {
			query:       tJSONQuery,
			inEvent:     newCloudEvent(t, tJSON1, cloudevents.ApplicationJSON),
			expectEvent: newCloudEvent(t, tJSONOutput1, cloudevents.ApplicationJSON),
		},
		"transform array ok": {
			query:       tJSONQuery2,
			inEvent:     newCloudEvent(t, tJSON2, cloudevents.ApplicationJSON),
			expectEvent: newCloudEvent(t, tJSONOutput2, cloudevents.ApplicationJSON),
		},
		"transform error": {
			query:       tJSONQuery,
			inEvent:     newCloudEvent(t, tInvalidJSON, cloudevents.ApplicationJSON),
			expectEvent: newCloudEvent(t, tInvalidJSONResponse, cloudevents.ApplicationJSON),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			env := &envAccessor{
				EnvConfig: adapter.EnvConfig{
					Component: tCloudEventSource,
				},
				Query: tc.query,
			}

			ceClient, send, responses := cetest.NewMockResponderClient(t, 1)

			a := NewAdapter(ctx, env, ceClient)

			go func() {
				if err := a.Start(ctx); err != nil {
					assert.FailNow(t, "could not start test adapter")
				}
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

func newCloudEvent(t *testing.T, data, contentType string, opts ...cloudEventOptions) cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetID(tCloudEventID)
	event.SetType(tCloudEventType)
	event.SetSource(tCloudEventSource)
	err := event.SetData(contentType, []byte(data))
	require.NoError(t, err)
	return event
}

func sendCE(t *testing.T, event *cloudevents.Event, sink string) protocol.Result {
	ctx := cloudevents.ContextWithTarget(context.Background(), sink)
	c, err := cloudevents.NewClientHTTP()
	require.NoError(t, err)

	result := c.Send(ctx, *event)
	return result
}
