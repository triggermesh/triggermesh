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

/*
Contains code excerpt based on VMware's VEBA's webhook

vCenter Event Broker Appliance
Copyright (c) 2019 VMware, Inc.  All rights reserved

The BSD-2 license (the "License") set forth below applies to all parts of the vCenter Event Broker Appliance project.  You may not use this file except in compliance with the License.

BSD-2 License

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package cloudeventssource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	zapt "go.uber.org/zap/zaptest"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cloudeventst "github.com/cloudevents/sdk-go/v2/client/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triggermesh/triggermesh/pkg/adapter/fs"
)

const (
	// path to secret fixtures.
	secret1Path = "../../../../test/fixtures/secrets/secret1"
	secret2Path = "../../../../test/fixtures/secrets/secret2"

	// values for secrets in the fixtures above.
	tSecret1 = "secret1"
	tSecret2 = "secret2"

	tUser  = "user"
	tToken = "token"
)

var (
	basicAuths KeyMountedValues = KeyMountedValues{
		{
			Key:              tUser,
			MountedValueFile: secret1Path,
		},
	}
	tokens KeyMountedValues = KeyMountedValues{
		{
			Key:              tToken,
			MountedValueFile: secret2Path,
		},
	}
)

func TestCloudEventsSource(t *testing.T) {
	logger := zapt.NewLogger(t).Sugar()

	successCE := cloudevents.NewEvent(cloudevents.VersionV1)
	successCE.SetType("type")
	successCE.SetSource("source")
	successCE.SetID("id")
	successCE.SetTime(time.Now())

	failCE := cloudevents.NewEvent(cloudevents.VersionV1)

	tc := map[string]struct {
		cloudEvent  cloudevents.Event
		expectError bool
	}{
		"success CE": {
			cloudEvent:  successCE,
			expectError: false,
		},
		"fail CE": {
			cloudEvent:  failCE,
			expectError: true,
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			cfw, err := fs.NewCachedFileWatcher(logger)
			require.NoError(t, err, "Could not create CachedFileWatcher")

			ceClient, chOut := cloudeventst.NewMockSenderClient(t, 1)

			handler := &cloudEventsHandler{
				cfw:      cfw,
				ceClient: ceClient,
				logger:   logger,
			}

			res := handler.handle(context.TODO(), c.cloudEvent)
			if c.expectError {
				require.False(t, cloudevents.IsACK(res), "Expected error handling CloudEvent did not happen")
				require.True(t, cloudevents.IsNACK(res), "Expected error handling CloudEvent did not happen")
				return
			} else {
				require.True(t, cloudevents.IsACK(res), "Unexpected error handling CloudEvent")
			}

			select {
			case event := <-chOut:
				assert.Equal(t, c.cloudEvent, event, "event Data does not match")

			case <-time.After(1 * time.Second):
				assert.Fail(t, "expected cloud event was not sent")
			}
		})
	}
}

func TestCloudEventsSourceAuthentication(t *testing.T) {
	logger := zapt.NewLogger(t).Sugar()

	tc := map[string]struct {
		requestUsername string
		requestPassword string
		requestHeaders  map[string]string

		expectCode int
	}{
		"no credentials sent": {
			expectCode: http.StatusUnauthorized,
		},
		"valid BasicAuth user": {
			requestUsername: tUser,
			requestPassword: tSecret1,
			expectCode:      http.StatusOK,
		},
		"wrong BasicAuth credentials, user does not exist": {
			requestUsername: tUser + "saltpepper",
			requestPassword: tSecret1,
			expectCode:      http.StatusUnauthorized,
		},
		"wrong BasicAuth credentials": {
			requestUsername: tUser,
			requestPassword: tSecret1 + "saltpepper",
			expectCode:      http.StatusUnauthorized,
		},
		"valid Token": {
			requestHeaders: map[string]string{tToken: tSecret2},
			expectCode:     http.StatusOK,
		},
		"wrong Token, header key is not used for authentication": {
			requestHeaders: map[string]string{tToken + "saltpepper": tSecret2},
			expectCode:     http.StatusUnauthorized,
		},
		"wrong Token value": {
			requestHeaders: map[string]string{tToken: tSecret2 + "saltpepper"},
			expectCode:     http.StatusUnauthorized,
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {

			cfw, err := fs.NewCachedFileWatcher(logger)
			require.NoError(t, err, "Could not create CachedFileWatcher")

			for _, path := range []string{secret1Path, secret2Path} {
				err := cfw.Add(path)
				require.NoError(t, err, "Could not set watch on secret path %s", path)
			}

			handler := &cloudEventsHandler{
				basicAuths: basicAuths,
				tokens:     tokens,

				cfw:    cfw,
				logger: logger,
			}

			h := handler.handleAuthentication(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			ts := httptest.NewServer(h)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL, nil)
			require.NoError(t, err, "Could not create test request")

			if c.requestUsername != "" {
				req.SetBasicAuth(c.requestUsername, c.requestPassword)
			}

			for k, v := range c.requestHeaders {
				req.Header.Add(k, v)
			}

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "There was an error testing the authentication handler")

			assert.Equal(t, c.expectCode, res.StatusCode, "Unexpected status code")
		})
	}
}
