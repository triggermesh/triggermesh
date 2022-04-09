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

package httppollersource

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cetest "github.com/cloudevents/sdk-go/v2/client/test"
	"github.com/stretchr/testify/assert"
	logtesting "knative.dev/pkg/logging/testing"
)

const (
	tContentType = "application/json"
	tEventType   = "test.eventype"
	tEventSource = "test.eventsource"
	tURLFailPath = "/fail"
)

func TestHTTPPollerRequests(t *testing.T) {

	// tResponse is used as payload for mocked responses.
	type tResponse struct {
		URL      string
		Method   string
		Headers  map[string]string
		Username string
		Password string
	}

	// tServer is the mocked server configured for tests.
	tServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log(r.URL.Path)
		if r.URL.Path == tURLFailPath {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", tContentType)
		w.WriteHeader(http.StatusOK)

		user, password, _ := r.BasicAuth()
		res := &tResponse{
			URL:      r.URL.String(),
			Method:   r.Method,
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

		if err := json.NewEncoder(w).Encode(res); err != nil {
			assert.FailNow(t, "mock service could not JSON encode response")
		}
	}))

	testCases := map[string]struct {
		// test request
		method            string
		headers           map[string]string
		basicAuthUsername string
		basicAuthPassword string

		// service mock induced failure
		fail bool
	}{
		"GET request": {
			method: "GET",
		},

		"POST request": {
			method: "POST",
		},

		"GET request with headers": {
			method: "GET",
			headers: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},

		"GET with basic auth request": {
			method:            "GET",
			basicAuthUsername: "jane",
			basicAuthPassword: "doe",
		},

		"GET failed request": {
			method: "GET",
			fail:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			httpClient := tServer.Client()
			ceClient, chEvent := cetest.NewMockSenderClient(t, 1,
				cloudevents.WithTimeNow(), cloudevents.WithUUIDs())

			tURL, err := url.Parse(tServer.URL)
			assert.NoError(t, err, "test URL is not valid")

			if tc.fail {
				tURL.Path = tURLFailPath
			}

			httpRequest, err := http.NewRequest(tc.method, tURL.String(), nil)
			if err != nil {
				assert.FailNow(t, "Cannot create request: %v", err)
			}

			for k, v := range tc.headers {
				httpRequest.Header.Set(k, v)
			}

			if tc.basicAuthUsername != "" || tc.basicAuthPassword != "" {
				httpRequest.SetBasicAuth(tc.basicAuthUsername, tc.basicAuthPassword)
			}

			p := httpPoller{
				eventType:   tEventType,
				eventSource: tEventSource,
				interval:    5 * time.Second,

				ceClient:    ceClient,
				httpRequest: httpRequest,
				httpClient:  httpClient,
				logger:      logtesting.TestLogger(t),
			}

			p.dispatch()

			select {
			case event := <-chEvent:
				if tc.fail {
					assert.Fail(t, "unexpected event received")
				}

				res := &tResponse{}
				assert.Equal(t, tEventType, event.Type(), "received unexpected event type")
				assert.Equal(t, tEventSource, event.Source(), "received unexpected event source")
				assert.NoError(t, event.DataAs(res), "error parsing response from mocked service")
				assert.Equal(t, tc.method, res.Method, "wrong HTTP method used")
				assert.Equal(t, tc.basicAuthUsername, res.Username, "wrong username")
				assert.Equal(t, tc.basicAuthPassword, res.Password, "wrong password")
				for k, v := range tc.headers {
					canonicalK := textproto.CanonicalMIMEHeaderKey(k)
					resv, ok := res.Headers[canonicalK]
					assert.True(t, ok, "header %q not found", canonicalK)
					assert.Equal(t, v, resv, "wrong value for header %q", canonicalK)
				}

			case <-time.After(1 * time.Second):
				if !tc.fail {
					assert.Fail(t, "expected cloud event containing was not sent")
				}
			}
		})
	}
}
