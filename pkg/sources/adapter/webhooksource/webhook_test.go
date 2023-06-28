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

package webhooksource

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/stretchr/testify/assert"

	zapt "go.uber.org/zap/zaptest"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cloudeventst "github.com/cloudevents/sdk-go/v2/client/test"
)

const (
	tEventType           = "testType"
	tEventSource         = "testSource"
	tHost                = "test-host"
	tResponseEventType   = "testRespType"
	tResponseEventSource = "testRespSource"
)

var (
	expectedExtensionsBase = map[string]interface{}{
		"host":   tHost,
		"method": http.MethodGet,
		"path":   "/",
	}
)

func TestWebhookEvent(t *testing.T) {

	logger := zapt.NewLogger(t).Sugar()

	tc := map[string]struct {
		body io.Reader

		username string
		password string
		query    string
		headers  map[string]string

		expectedCode             int
		expectedResponseContains string
		expectedEventData        string
		expectedExtensions       map[string]interface{}
		responseResult           protocol.Result
		responseEvent            *event.Event
	}{
		"nil body": {
			body: nil,

			expectedCode:             http.StatusBadRequest,
			expectedResponseContains: "request without body not supported",
			expectedExtensions:       expectedExtensionsBase,
		},

		"arbitrary message": {
			body: read("arbitrary message"),

			expectedCode:       http.StatusOK,
			expectedEventData:  "arbitrary message",
			expectedExtensions: expectedExtensionsBase,
		},

		"basic auth no header": {
			body:     read("arbitrary message"),
			username: "foo",
			password: "bar",

			expectedCode:             http.StatusBadRequest,
			expectedResponseContains: "wrong authentication header",
			expectedExtensions:       expectedExtensionsBase,
		},

		"basic auth wrong header": {
			body: read("arbitrary message"),
			headers: map[string]string{
				"Authorization": "wrong auth",
			},

			username: "foo",
			password: "bar",

			expectedCode:             http.StatusBadRequest,
			expectedResponseContains: "wrong authentication header",
			expectedExtensions:       expectedExtensionsBase,
		},

		"basic auth wrong creds": {
			body: read("arbitrary message"),
			headers: map[string]string{
				"Authorization": basicAuth("boo", "far"),
			},

			username: "foo",
			password: "bar",

			expectedCode:             http.StatusUnauthorized,
			expectedResponseContains: "credentials are not valid",
			expectedExtensions:       expectedExtensionsBase,
		},

		"basic auth success": {
			body: read("arbitrary message"),
			headers: map[string]string{
				"Authorization": basicAuth("foo", "bar"),
			},

			username: "foo",
			password: "bar",

			expectedCode:       http.StatusOK,
			expectedEventData:  "arbitrary message",
			expectedExtensions: expectedExtensionsBase,
		},

		"extra headers": {
			body: read("arbitrary message"),
			headers: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},

			expectedCode:      http.StatusOK,
			expectedEventData: "arbitrary message",
			expectedExtensions: expectedExtensions(map[string]string{
				"hk1": "v1",
				"hk2": "v2",
			}),
		},

		"extra queries": {
			body:  read("arbitrary message"),
			query: "?k1=v1&k2=v2",

			expectedCode:      http.StatusOK,
			expectedEventData: "arbitrary message",
			expectedExtensions: expectedExtensions(map[string]string{
				"qk1": "v1",
				"qk2": "v2",
			}),
		},

		"empty response": {
			body:          read("arbitrary message"),
			responseEvent: newEvent(""),

			expectedCode:       http.StatusNoContent,
			expectedEventData:  "arbitrary message",
			expectedExtensions: expectedExtensionsBase,
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			replierFn := func(inMessage event.Event) (*event.Event, protocol.Result) {
				// If the test case does not define a response event use a default one
				e := c.responseEvent
				if e == nil {
					e = newEvent(`{"test":"default"}`)
				}

				// If the test case does not define a result return ACK
				r := c.responseResult
				if r == nil {
					r = protocol.ResultACK
				}
				return e, r
			}
			ceClient, chEvent := cloudeventst.NewMockRequesterClient(t, 1, replierFn, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
			handler := &webhookHandler{
				eventType:   tEventType,
				eventSource: tEventSource,
				username:    c.username,
				password:    c.password,

				ceClient: ceClient,
				logger:   logger,
				extensionAttributesFrom: &ExtensionAttributesFrom{
					method:  true,
					path:    true,
					host:    true,
					queries: true,
					headers: true,
				},
			}

			req, _ := http.NewRequest(http.MethodGet, "/"+c.query, c.body)
			for k, v := range c.headers {
				req.Header.Add(k, v)
			}
			req.Host = tHost

			ctx := context.Background()

			th := http.HandlerFunc(handler.handleAll(ctx))

			rr := httptest.NewRecorder()

			th.ServeHTTP(rr, req)

			assert.Equal(t, c.expectedCode, rr.Code, "unexpected response code")
			assert.Contains(t, rr.Body.String(), c.expectedResponseContains, "could not find expected response")
			if c.expectedEventData != "" {
				select {
				case event := <-chEvent:
					assert.Equal(t, c.expectedEventData, string(event.Data()), "event Data does not match")
					assert.Equal(t, c.expectedExtensions, event.Context.GetExtensions(), "event extensions does not match")

				case <-time.After(1 * time.Second):
					assert.Fail(t, "expected cloud event containing %q was not sent", c.expectedEventData)
				}
			}
		})
	}
}

func read(s string) io.Reader {
	return strings.NewReader(s)
}

func basicAuth(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

func TestAttributeSanitize(t *testing.T) {
	tc := map[string]struct {
		name      string
		sanitized string
	}{
		"no change": {
			name:      "myattribute",
			sanitized: "myattribute",
		},
		"truncate more than 20 chars": {
			name:      "123456789012345678901",
			sanitized: "12345678901234567890",
		},
		"upper case": {
			name:      "1A2B3c4d",
			sanitized: "1a2b3c4d",
		},
		"non valid chars": {
			name:      "*-?*abcd",
			sanitized: "abcd",
		},
		"reserved word data": {
			name:      "data",
			sanitized: "data0",
		},
		"reserved word data upper case": {
			name:      "DatA",
			sanitized: "data0",
		},
		"more than 20 chars, some non valid, some upper case": {
			name:      "1234567890*?'abcdeº!FGHIJxxxx?",
			sanitized: "1234567890abcdefghij",
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			r := sanitizeCloudEventAttributeName(c.name)
			assert.Equal(t, c.sanitized, r)
		})
	}
}

func expectedExtensions(extensions map[string]string) map[string]interface{} {
	ee := make(map[string]interface{}, len(expectedExtensionsBase)+len(extensions))
	// copy the base expected extensions
	for k, v := range expectedExtensionsBase {
		ee[k] = v
	}
	// extend or overwrite with the extensions provided for the test
	for k, v := range extensions {
		ee[k] = v
	}
	return ee
}

func newEvent(body string) *event.Event {
	e := event.New(event.CloudEventsVersionV1)
	e.SetType(tResponseEventType)
	e.SetSource(tResponseEventSource)

	if body != "" {
		e.SetData("text/json", body)
	}

	return &e
}
