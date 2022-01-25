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

package dataweavetransformation

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

	tJSON1       = `[{"name": "User1","age": 19},{"name": "User2","age": 18},{"name": "User3","age": 15},{"name": "User4","age": 13},{"name": "User5","age": 16}]`
	tSpell1      = "output application/json --- payload filter (item) -> item.age > 15"
	tJSONOutput1 = `[{"name": "User1","age": 19},{"name": "User2","age": 18},{"name": "User3","age": 15},{"name": "User4","age": 13},{"name": "User5","age": 16}]`

	tFalseJson         = `"this is not JSON"`
	tFalseJSONResponse = "{\"Code\":\"adapter-process\",\"Description\":\"exit status 255\",\"Details\":\"executing the spell\"}"
	tReplierResponse   = "\"[  {    \\\"name\\\": \\\"User1\\\",    \\\"age\\\": 19  },  {    \\\"name\\\": \\\"User2\\\",    \\\"age\\\": 18  },  {    \\\"name\\\": \\\"User5\\\",    \\\"age\\\": 16  }]\""
)

func TestSink(t *testing.T) {
	testCases := map[string]struct {
		inEvent     cloudevents.Event
		expectEvent cloudevents.Event
		spell       string
	}{
		"sink ok": {
			inEvent:     newCloudEvent(t, tJSON1, cloudevents.ApplicationJSON),
			expectEvent: newCloudEvent(t, tJSONOutput1, cloudevents.ApplicationJSON),
			spell:       tSpell1,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := ioutil.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Equal(t, tJSONOutput1, string(body))
				fmt.Fprintf(w, "OK")
			}))
			defer svr.Close()

			env := &envAccessor{
				EnvConfig: adapter.EnvConfig{
					Component: tCloudEventSource,
					Sink:      svr.URL,
				},
				Spell:               tc.spell,
				IncomingContentType: cloudevents.ApplicationJSON,
			}
			ctx := context.Background()
			c, err := cloudevents.NewClientHTTP()
			assert.NoError(t, err)
			a := NewAdapter(ctx, env, c)

			go func() {
				if err := a.Start(ctx); err != nil {
					assert.NoError(t, err)
				}
			}()

			response := sendCE(t, &tc.inEvent, c, svr.URL)
			assert.NotEqual(t, cloudevents.IsUndelivered(response), response)
		})
	}
}

func TestReplier(t *testing.T) {
	testCases := map[string]struct {
		inEvent     cloudevents.Event
		expectEvent cloudevents.Event
		spell       string
	}{
		"transform ok": {
			inEvent:     newCloudEvent(t, tJSON1, cloudevents.ApplicationJSON),
			expectEvent: newCloudEvent(t, tReplierResponse, cloudevents.ApplicationJSON),
			spell:       tSpell1,
		},
		"transform error": {
			inEvent:     newCloudEvent(t, tFalseJson, cloudevents.ApplicationJSON),
			expectEvent: newCloudEvent(t, tFalseJSONResponse, cloudevents.ApplicationXML),
			spell:       tSpell1,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			env := &envAccessor{
				EnvConfig: adapter.EnvConfig{
					Component: tCloudEventSource,
				},
				Spell:               tc.spell,
				IncomingContentType: cloudevents.ApplicationJSON,
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

func sendCE(t *testing.T, event *cloudevents.Event, cs cloudevents.Client, sink string) protocol.Result {
	ctx := cloudevents.ContextWithTarget(context.Background(), sink)
	c, err := cloudevents.NewClientHTTP()
	require.NoError(t, err)

	result := c.Send(ctx, *event)
	return result
}
