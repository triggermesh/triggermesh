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

package slack

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestWebAPI(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	testCases := map[string]struct {
		methodURL string
		data      string

		mockStatus   int
		mockResponse string

		expectOK      bool
		expectStatus  int
		expectError   string
		expectWarning string
	}{
		"post message invalid response": {
			methodURL: "chat.postMessage",
			data:      `{"channel":"C01112A09FT", "text": "Hello from TriggerMesh!"}`,

			mockStatus:   200,
			mockResponse: "This is not JSON",

			expectOK:     false,
			expectStatus: 200,
			expectError:  "invalid character 'T' looking for beginning of value",
		},

		"post message failed": {
			methodURL: "chat.postMessage",
			data:      `{"channel":"C01112A09FT", "text": "Hello from TriggerMesh!"}`,

			mockStatus:   200,
			mockResponse: `{"ok":false,"error":"could not send message"}`,

			expectOK:     false,
			expectStatus: 200,
			expectError:  "could not send message",
		},

		"post message failed with no ok message": {
			methodURL: "chat.postMessage",
			data:      `{"channel":"C01112A09FT", "text": "Hello from TriggerMesh!"}`,

			mockStatus:   200,
			mockResponse: `{"error":"could not send message"}`,

			expectOK:     false,
			expectStatus: 200,
			expectError:  "could not send message",
		},

		"post message": {
			methodURL: "chat.postMessage",
			data:      `{"channel":"C01112A09FT", "text": "Hello from TriggerMesh!"}`,

			mockStatus:   200,
			mockResponse: `{"ok":true,"channel":"C01112A09FT","ts":"1593446134.003900","message":{"bot_id":"B01628BCTMZ","type":"message","text":"Hello from TriggerMesh!","user":"U016RST62SU","ts":"1593446134.003900","team":"TA1J7JEBS"}}`,

			expectOK:     true,
			expectStatus: 200,
		},

		"post message with warning": {
			methodURL: "chat.postMessage",
			data:      `{"channel":"C01112A09FT", "text": "Hello from TriggerMesh!"}`,

			mockStatus:   200,
			mockResponse: `{"ok":true, "warning": "channel being deleted", "channel":"C01112A09FT","ts":"1593446134.003900","message":{"bot_id":"B01628BCTMZ","type":"message","text":"Hello from TriggerMesh!","user":"U016RST62SU","ts":"1593446134.003900","team":"TA1J7JEBS"}}`,

			expectOK:      true,
			expectStatus:  200,
			expectWarning: "channel being deleted",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			response := httpmock.NewStringResponder(tc.mockStatus, tc.mockResponse)
			mockURL := fmt.Sprintf("http://mocked/api/%s", tc.methodURL)
			httpmock.RegisterResponder("POST", mockURL, response)

			client := NewWebAPIClient("token", "http://mocked/api/", &http.Client{}, GetFullCatalog(true))
			r, err := client.Do(tc.methodURL, []byte(tc.data))

			assert.NoError(t, err)
			assert.Equal(t, r.IsOK(), tc.expectOK)
			assert.Equal(t, r.StatusCode(), tc.expectStatus)
			assert.Equal(t, r.Error(), tc.expectError)
			assert.Equal(t, r.Warning(), tc.expectWarning)

			if err != nil {
				t.Logf("error: %+v", err)
			}
			t.Logf("response: %+v", r)
		})
	}
}
