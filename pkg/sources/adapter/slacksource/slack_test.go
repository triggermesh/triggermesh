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

package slacksource

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cloudeventst "github.com/cloudevents/sdk-go/v2/client/test"
	"github.com/stretchr/testify/assert"
	zapt "go.uber.org/zap/zaptest"
)

func TestSlackEvent(t *testing.T) {

	logger := zapt.NewLogger(t).Sugar()

	tc := map[string]struct {
		body          io.Reader
		headers       map[string]string
		appID         string
		signingSecret string
		timewrap      timeWrap

		expectedCode     int
		expectedContains string

		expectedEventID   string
		expectedEventData string
	}{
		"nil body": {
			body: nil,

			expectedCode:     http.StatusBadRequest,
			expectedContains: "request without body not supported",
		},

		"not a JSON message": {
			body: read("this is not an expected message"),

			expectedCode:     http.StatusBadRequest,
			expectedContains: "could not unmarshal JSON request:",
		},

		"not an expected message": {
			body: read(`{"hello":"world"}`),

			expectedCode: http.StatusOK,
		},

		"url verification": {
			body: read(`
			{
				"token": "XXYYZZ",
				"challenge": "3eZbrw1aBm2rZgRNFdxV2595E9CY3gmdALWMmHkvFXO7tYXAYM8P",
				"type": "url_verification"
			}`),

			expectedCode:     http.StatusOK,
			expectedContains: `{"challenge":"3eZbrw1aBm2rZgRNFdxV2595E9CY3gmdALWMmHkvFXO7tYXAYM8P"}`,
		},

		"wrong App ID": {
			body: read(`
			{
		    "token": "XXYYZZ",
		    "team_id": "TXXXXXXXX",
		    "api_app_id": "AXXXXXXXXX",
		    "type": "event_callback",
		    "event_id": "Ev08MFMKH6",
		    "event_time": 1234567890
			}`),
			appID: "ZYYYYYYYYYY",

			expectedCode: http.StatusOK,
		},

		"handle callback": {
			body: read(`
			{
		    "token": "XXYYZZ",
		    "team_id": "TXXXXXXXX",
		    "api_app_id": "AXXXXXXXXX",
		    "event": {
		            "type": "name_of_event",
		            "event_ts": "1234567890.123456",
		            "user": "UXXXXXXX1"
		    },
		    "type": "event_callback",
		    "authed_users": [
		            "UXXXXXXX1",
		            "UXXXXXXX2"
		    ],
		    "event_id": "Ev08MFMKH6",
		    "event_time": 1234567890
			}`),

			expectedCode:      http.StatusOK,
			expectedEventID:   "Ev08MFMKH6",
			expectedEventData: `{"event_ts":"1234567890.123456","type":"name_of_event","user":"UXXXXXXX1"}`,
		},

		"missing signing secret": {
			body: read(`
			{
		    "token": "XXYYZZ",
		    "team_id": "TXXXXXXXX",
		    "api_app_id": "AXXXXXXXXX",
		    "type": "event_callback",
		    "event_id": "Ev08MFMKH6",
		    "event_time": 1234567890
			}`),
			signingSecret: "6623e5d64e469c64908c481b6de975f0",

			expectedCode:     http.StatusUnauthorized,
			expectedContains: "empty signature header",
		},

		"signature does not begin with v0=": {
			body: read(`
			{
		    "token": "XXYYZZ",
		    "team_id": "TXXXXXXXX",
		    "api_app_id": "AXXXXXXXXX",
		    "type": "event_callback",
		    "event_id": "Ev08MFMKH6",
		    "event_time": 1234567890
			}`),
			signingSecret: "6623e5d64e469c64908c481b6de975f0",
			headers:       map[string]string{signatureHeader: "x2=aaa"},

			expectedCode:     http.StatusUnauthorized,
			expectedContains: `signature header format does not begin with "v0=": x2=aaa`,
		},

		"missing signing timestamp": {
			body: read(`
			{
		    "token": "XXYYZZ",
		    "team_id": "TXXXXXXXX",
		    "api_app_id": "AXXXXXXXXX",
		    "type": "event_callback",
		    "event_id": "Ev08MFMKH6",
		    "event_time": 1234567890
			}`),
			signingSecret: "6623e5d64e469c64908c481b6de975f0",
			headers:       map[string]string{signatureHeader: "v0=aaa"},

			expectedCode:     http.StatusUnauthorized,
			expectedContains: "empty signature timestamp header",
		},

		"wrong signing timestamp": {
			body: read(`
			{
		    "token": "XXYYZZ",
		    "team_id": "TXXXXXXXX",
		    "api_app_id": "AXXXXXXXXX",
		    "type": "event_callback",
		    "event_id": "Ev08MFMKH6",
		    "event_time": 1234567890
			}`),
			signingSecret: "6623e5d64e469c64908c481b6de975f0",
			headers: map[string]string{
				signatureHeader:          "v0=aaa",
				signatureTimestampHeader: "abcde",
			},

			expectedCode:     http.StatusUnauthorized,
			expectedContains: "error parsing header timestamp",
		},

		"expired signing timestamp": {
			body: read(`
			{
		    "token": "XXYYZZ",
		    "team_id": "TXXXXXXXX",
		    "api_app_id": "AXXXXXXXXX",
		    "type": "event_callback",
		    "event_id": "Ev08MFMKH6",
		    "event_time": 1234567890
			}`),
			signingSecret: "6623e5d64e469c64908c481b6de975f0",
			headers: map[string]string{
				signatureHeader:          "v0=aaa",
				signatureTimestampHeader: "1593192795",
			},

			expectedCode:     http.StatusUnauthorized,
			expectedContains: "signing timestamp expired",
		},

		"signed event ok": {
			body:          read(`{"token":"rryC9de5GMzbu8oA3qVZRcVY","team_id":"TA1J7JEBS","api_app_id":"A01624EULRY","event":{"client_msg_id":"ed372e63-845a-4981-9c3d-ecb8f64c6cef","type":"message","text":"<@U016RST62SU> asdfa","user":"UT8LFLXR8","ts":"1593192794.008000","team":"TA1J7JEBS","blocks":[{"type":"rich_text","block_id":"R5h","elements":[{"type":"rich_text_section","elements":[{"type":"user","user_id":"U016RST62SU"},{"type":"text","text":" asdfa"}]}]}],"channel":"C01112A09FT","event_ts":"1593192794.008000","channel_type":"channel"},"type":"event_callback","event_id":"Ev016HBFLLP3","event_time":1593192794,"authed_users":["U015MC994F9"]}`),
			signingSecret: "6623e5d64e469c64908c481b6de975f0",
			headers: map[string]string{
				signatureHeader:          "v0=e270150d7eee7a9f176e056f4b647ea4f529e5b4946de36e82e29be6364a72a9",
				signatureTimestampHeader: "1593192795",
			},
			timewrap: &mockedTime{
				time.Unix(1593192796, 0),
			},

			expectedCode: http.StatusOK,
		},

		"signed event tampered": {
			body:          read(`{"TamPeREd":"yES","token":"rryC9de5GMzbu8oA3qVZRcVY","team_id":"TA1J7JEBS","api_app_id":"A01624EULRY","event":{"client_msg_id":"ed372e63-845a-4981-9c3d-ecb8f64c6cef","type":"message","text":"<@U016RST62SU> asdfa","user":"UT8LFLXR8","ts":"1593192794.008000","team":"TA1J7JEBS","blocks":[{"type":"rich_text","block_id":"R5h","elements":[{"type":"rich_text_section","elements":[{"type":"user","user_id":"U016RST62SU"},{"type":"text","text":" asdfa"}]}]}],"channel":"C01112A09FT","event_ts":"1593192794.008000","channel_type":"channel"},"type":"event_callback","event_id":"Ev016HBFLLP3","event_time":1593192794,"authed_users":["U015MC994F9"]}`),
			signingSecret: "6623e5d64e469c64908c481b6de975f0",
			headers: map[string]string{
				signatureHeader:          "v0=e270150d7eee7a9f176e056f4b647ea4f529e5b4946de36e82e29be6364a72a9",
				signatureTimestampHeader: "1593192795",
			},
			timewrap: &mockedTime{
				time.Unix(1593192796, 0),
			},

			expectedCode:     http.StatusUnauthorized,
			expectedContains: "received wrong signature signing hash",
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			ceClient, chEvent := cloudeventst.NewMockSenderClient(t, 1)

			tw := c.timewrap
			if tw == nil {
				tw = standardTime{}
			}

			handler := &slackEventAPIHandler{
				appID:         c.appID,
				signingSecret: c.signingSecret,
				ceClient:      ceClient,
				logger:        logger,
				time:          tw,
			}

			req, _ := http.NewRequest("GET", "/", c.body)
			for k, v := range c.headers {
				req.Header.Add(k, v)
			}

			rr := httptest.NewRecorder()
			th := http.HandlerFunc(handler.handleAll)

			th.ServeHTTP(rr, req)

			assert.Equal(t, c.expectedCode, rr.Code, "unexpected response code")
			assert.Contains(t, rr.Body.String(), c.expectedContains, "could not find expected response")

			if c.expectedEventID != "" {
				select {
				case event := <-chEvent:
					assert.Equal(t, c.expectedEventID, event.ID(), "event ID does not match")
					assert.Equal(t, c.expectedEventData, string(event.Data()), "event Data does not match")

				case <-time.After(1 * time.Second):
					assert.Fail(t, "expected cloud event by ID %q was not sent", c.expectedEventID)
				}
			}
		})
	}
}

type mockedTime struct {
	t time.Time
}

func (s *mockedTime) Now() time.Time {
	return s.t
}

var _ timeWrap = (*mockedTime)(nil)

func read(s string) io.Reader {
	return strings.NewReader(s)
}
