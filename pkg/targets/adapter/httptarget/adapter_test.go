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

package httptarget

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"testing"
	"time"

	cetest "github.com/cloudevents/sdk-go/v2/client/test"
	ceevent "github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"

	logtesting "knative.dev/pkg/logging/testing"
)

const (
	tURL         = "http://mytests"
	tEventType   = "test.eventype"
	tEventSource = "test.eventsource"

	tID          = "abc123"
	tContentType = "application/json"
	tCEType      = "io.triggermesh.http.request"
	tCESource    = "test.source"

	tCETypeArbitrary = "test.some.type"
)

func TestHTTPRequests(t *testing.T) {
	type tRequest struct {
		Status int
	}

	type tResponse struct {
		Query    string
		Method   string
		Path     string
		Headers  map[string]string
		Username string
		Password string
	}

	tServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := &tRequest{}
		err := json.NewDecoder(r.Body).Decode(req)
		if err != nil {
			assert.FailNow(t, "mock service reveived a not valid request")
		}

		w.Header().Set("Content-Type", "application/json")

		if req.Status == 0 {
			assert.FailNow(t, "mock service received a request with an invalid status code")
		}
		w.WriteHeader(req.Status)

		user, password, _ := r.BasicAuth()
		res := &tResponse{
			Query:    r.URL.RawQuery,
			Method:   r.Method,
			Path:     r.URL.Path,
			Username: user,
			Password: password,
		}

		if len(r.Header) != 0 {
			res.Headers = make(map[string]string, len(r.Header))
			for k, v := range r.Header {
				if len(v) == 0 {
					res.Headers[k] = ""
					continue
				}
				res.Headers[k] = v[0]
			}
		}

		if err = json.NewEncoder(w).Encode(res); err != nil {
			assert.FailNow(t, "mock service could not JSON encode response")
		}
	}))

	testCases := map[string]struct {
		// adapter config
		url               string
		method            string
		skipVerify        bool
		caCertificate     string
		headers           map[string]string
		basicAuthUsername string
		basicAuthPassword string

		// service mock
		status int

		// request
		query      string
		pathsuffix string
		reqHeaders map[string]string

		// expected
		expectQuery string
		expectPath  string
	}{
		"GET request": {
			url:    tURL,
			method: "GET",

			status: 200,

			expectPath: "/",
		},

		"GET with query": {
			url:    tURL,
			method: "GET",

			status: 200,
			query:  "a=1",

			expectPath:  "/",
			expectQuery: "a=1",
		},

		"GET with path suffix": {
			url:    tURL,
			method: "GET",

			status:     200,
			pathsuffix: "test/path",

			expectPath: "/test/path",
		},

		"simple GET with headers": {
			url:    tURL,
			method: "GET",
			headers: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},

			status: 200,

			expectPath: "/",
		},

		"GET with 201 status": {
			url:    tURL,
			method: "GET",

			status: 201,

			expectPath: "/",
		},

		"POST request": {
			url:    tURL,
			method: "POST",

			status: 200,

			expectPath: "/",
		},

		"basic auth request": {
			url:               tURL,
			method:            "GET",
			basicAuthUsername: "jane",
			basicAuthPassword: "doe",

			status: 200,

			expectPath: "/",
		},

		"GET request with headers": {
			url:        tURL,
			method:     "GET",
			reqHeaders: map[string]string{"Authorization": "Bearer abc123"},

			status: 200,

			expectPath: "/",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := tServer.Client()

			ceClient, send, responses := cetest.NewMockResponderClient(t, 1)

			u, err := url.Parse(tServer.URL)
			assert.NoError(t, err, "test URL is not valid")

			adapter := &httpAdapter{
				eventType:   tEventType,
				eventSource: tEventSource,

				url:               u,
				method:            tc.method,
				headers:           tc.headers,
				basicAuthUsername: tc.basicAuthUsername,
				basicAuthPassword: tc.basicAuthPassword,
				client:            client,

				ceClient: ceClient,
				logger:   logtesting.TestLogger(t),
			}

			go func() {
				if err := adapter.Start(context.Background()); err != nil {
					assert.FailNow(t, "could not start test adapter")
				}
			}()

			req := &tRequest{
				Status: tc.status,
			}
			body, err := json.Marshal(req)
			if err != nil {
				assert.FailNow(t, "could not marshal JSON request: %v", req)
			}

			rd := RequestData{
				Query:      tc.query,
				PathSuffix: tc.pathsuffix,
				Body:       body,
				Headers:    tc.reqHeaders,
			}

			event := ceevent.New()
			if err = event.SetData(tContentType, rd); err != nil {
				assert.Fail(t, "could not write test payload to CloudEvent: %v", rd)
			}

			event.SetID(tID)
			event.SetType(tCEType)
			event.SetSource(tCESource)

			send <- event

			select {
			case event := <-responses:
				res := &tResponse{}
				assert.NoError(t, event.Event.DataAs(res), "error parsing response from mocked service")
				assert.Equal(t, tc.method, res.Method, "wrong HTTP method used")
				assert.Equal(t, tc.expectQuery, res.Query, "wrong HTTP query string")
				assert.Equal(t, tc.expectPath, res.Path, "wrong HTTP path")
				assert.Equal(t, tc.basicAuthUsername, res.Username, "wrong username")
				assert.Equal(t, tc.basicAuthPassword, res.Password, "wrong password")
				for k, v := range tc.headers {
					canonicalK := textproto.CanonicalMIMEHeaderKey(k)
					resv, ok := res.Headers[canonicalK]
					assert.True(t, ok, "header %q not found", canonicalK)
					assert.Equal(t, v, resv, "wrong value for header %q", canonicalK)
				}
				for k, v := range tc.reqHeaders {
					canonicalK := textproto.CanonicalMIMEHeaderKey(k)
					resv, ok := res.Headers[canonicalK]
					assert.True(t, ok, "header %q not found", canonicalK)
					assert.Equal(t, v, resv, "wrong value for header %q", canonicalK)
				}

			case <-time.After(1 * time.Second):
				assert.Fail(t, "expected cloud event response was not received")
			}
		})
	}
}

func TestArbitraryEventTypeHTTPRequest(t *testing.T) {
	type tResponse struct {
		Method   string
		Path     string
		Headers  map[string]string
		Username string
		Password string
		Body     []byte
	}

	tServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			assert.FailNow(t, "mock service could not read body from request")
		}

		user, password, _ := r.BasicAuth()
		res := &tResponse{
			Method:   r.Method,
			Path:     r.URL.Path,
			Username: user,
			Password: password,
			Body:     body,
		}

		if len(r.Header) != 0 {
			res.Headers = make(map[string]string, len(r.Header))
			for k, v := range r.Header {
				if len(v) == 0 {
					res.Headers[k] = ""
					continue
				}
				res.Headers[k] = v[0]
			}
		}

		if err = json.NewEncoder(w).Encode(res); err != nil {
			assert.FailNow(t, "mock service could not JSON encode response")
		}
	}))

	testCases := map[string]struct {
		// adapter config

		url               string
		method            string
		skipVerify        bool
		caCertificate     string
		headers           map[string]string
		basicAuthUsername string
		basicAuthPassword string

		// service mock
		status int

		// request
		body []byte
	}{
		"POST request": {
			url:    tURL,
			method: "POST",

			status: 200,
			body:   []byte("hello world"),
		},

		"simple POST with headers": {
			url:    tURL,
			method: "POST",
			headers: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			body: []byte(`{"hello":"world"}`),

			status: 200,
		},

		"basic auth request": {
			url:               tURL,
			method:            "GET",
			basicAuthUsername: "jane",
			basicAuthPassword: "doe",
			body:              []byte(`{"hello":"world"}`),

			status: 200,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client := tServer.Client()

			ceClient, send, responses := cetest.NewMockResponderClient(t, 1)

			u, err := url.Parse(tServer.URL)
			assert.NoError(t, err, "test URL is not valid")

			adapter := &httpAdapter{
				eventType:   tEventType,
				eventSource: tEventSource,

				url:               u,
				method:            tc.method,
				headers:           tc.headers,
				basicAuthUsername: tc.basicAuthUsername,
				basicAuthPassword: tc.basicAuthPassword,
				client:            client,

				ceClient: ceClient,
				logger:   logtesting.TestLogger(t),
			}

			go func() {
				if err := adapter.Start(context.Background()); err != nil {
					assert.FailNow(t, "could not start test adapter")
				}
			}()

			event := ceevent.New()
			if err = event.SetData(tContentType, tc.body); err != nil {
				assert.Fail(t, "could not write test payload to CloudEvent: %v", string(tc.body))
			}

			event.SetID(tID)
			event.SetType(tCETypeArbitrary)
			event.SetSource(tCESource)

			send <- event

			select {
			case event := <-responses:
				res := &tResponse{}
				assert.NoError(t, event.Event.DataAs(res), "error parsing response from mocked service")
				assert.Equal(t, tc.method, res.Method, "wrong HTTP method used")
				assert.Equal(t, tc.basicAuthUsername, res.Username, "wrong username")
				assert.Equal(t, tc.basicAuthPassword, res.Password, "wrong password")
				assert.Equal(t, tc.body, res.Body, "unexpected body")
				for k, v := range tc.headers {
					canonicalK := textproto.CanonicalMIMEHeaderKey(k)
					resv, ok := res.Headers[canonicalK]
					assert.True(t, ok, "header %q not found", canonicalK)
					assert.Equal(t, v, resv, "wrong value for header %q", canonicalK)
				}

			case <-time.After(1 * time.Second):
				assert.Fail(t, "expected cloud event response was not received")
			}
		})
	}
}
