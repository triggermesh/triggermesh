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

	"github.com/stretchr/testify/assert"

	zapt "go.uber.org/zap/zaptest"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cloudeventst "github.com/cloudevents/sdk-go/v2/client/test"
)

const (
	tEventType   = "testType"
	tEventSource = "testSource"
)

func TestWebhookEvent(t *testing.T) {

	logger := zapt.NewLogger(t).Sugar()

	tc := map[string]struct {
		body io.Reader

		username string
		password string
		headers  map[string]string

		expectedCode             int
		expectedResponseContains string
		expectedEventData        string
	}{
		"nil body": {
			body: nil,

			expectedCode:             http.StatusBadRequest,
			expectedResponseContains: "request without body not supported",
		},

		"arbitrary message": {
			body: read("arbitrary message"),

			expectedCode:      http.StatusOK,
			expectedEventData: "arbitrary message",
		},

		"basic auth no header": {
			body:     read("arbitrary message"),
			username: "foo",
			password: "bar",

			expectedCode:             http.StatusBadRequest,
			expectedResponseContains: "wrong authentication header",
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
		},

		"basic auth success": {
			body: read("arbitrary message"),
			headers: map[string]string{
				"Authorization": basicAuth("foo", "bar"),
			},

			username: "foo",
			password: "bar",

			expectedCode: http.StatusOK,
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			ceClient, chEvent := cloudeventst.NewMockSenderClient(t, 1,
				cloudevents.WithTimeNow(), cloudevents.WithUUIDs(),
			)

			handler := &webhookHandler{
				eventType:   tEventType,
				eventSource: tEventSource,
				username:    c.username,
				password:    c.password,

				ceClient: ceClient,
				logger:   logger,
			}

			req, _ := http.NewRequest("GET", "/", c.body)
			for k, v := range c.headers {
				req.Header.Add(k, v)
			}

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
