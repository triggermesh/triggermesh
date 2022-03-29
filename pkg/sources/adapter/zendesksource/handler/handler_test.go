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

package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	logtesting "knative.dev/pkg/logging/testing"
)

const (
	tTicketID   = "0"
	tTicketType = "Incident"
	tEventSrc   = "test.source"
	tB64UsrPass = "YWRtaW46aHVudGVyMg=="
)

func TestHandler(t *testing.T) {
	t.Run("unsupported method", func(t *testing.T) {
		h := newTestHandler(t)

		req, err := http.NewRequest(http.MethodGet, "", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	})

	t.Run("unsupported content type", func(t *testing.T) {
		h := newTestHandler(t)

		msgBody := strings.NewReader(tTicketCreated)
		req := newPostRequest(t, msgBody)

		req.Header.Set(headerContentTypeKey, "not/supported; charset=utf-8")

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
	})

	t.Run("valid ticket created event", func(t *testing.T) {
		ceClient := adaptertest.NewTestClient()

		h := newTestHandler(t)
		h.ceClient = ceClient

		msgBody := strings.NewReader(tTicketCreated)
		req := newPostRequest(t, msgBody)

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		sentEvents := ceClient.Sent()
		require.Len(t, sentEvents, 1)

		event := sentEvents[0]
		assert.Equal(t, "com.zendesk.ticket.created", event.Type())
		assert.Equal(t, tEventSrc, event.Source())
		assert.Equal(t, tTicketID, event.Subject())
		assert.Equal(t, tTicketType, event.Extensions()[ceExtTicketType])
	})

	t.Run("invalid auth header", func(t *testing.T) {
		h := newTestHandler(t)

		msgBody := strings.NewReader("")
		req := newPostRequest(t, msgBody)

		req.Header.Set(headerAuthKey, "invalid")

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("invalid event", func(t *testing.T) {
		ceClient := adaptertest.NewTestClient()

		h := newTestHandler(t)
		h.ceClient = ceClient

		msgBody := strings.NewReader("{ not a JSON }")

		req := newPostRequest(t, msgBody)

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Failed to parse event data:")

		require.Empty(t, ceClient.Sent())
	})
}

func newTestHandler(t *testing.T) *Handler {
	t.Helper()

	return &Handler{
		logger:        logtesting.TestLogger(t),
		eventSrc:      tEventSrc,
		base64UsrPass: tB64UsrPass,
	}
}

func newPostRequest(t *testing.T, body io.Reader) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, "", body)
	require.NoError(t, err)

	req.Header.Set(headerContentTypeKey, "application/json; charset=utf-8")
	req.Header.Set(headerAuthKey, "Basic "+tB64UsrPass)

	return req
}

const tTicketCreated = `{
  "ticket": {
    "id": ` + tTicketID + `,
    "external_id": "ahg35h3jh",
    "title": "Help, my printer is on fire!",
    "url": "https://company.zendesk.com/api/v2/tickets/00000.json",
    "description": "The fire is very colorful.",
    "via": "Web Form",
    "status": "Open",
    "priority": "High",
    "ticket_type": "` + tTicketType + `",
    "group_name": "Support",
    "brand_name": "ACME",
    "due_date": "",
    "account": "ACME",
    "assignee": {
      "email": "jane@example.com",
      "name": "Jane Doe",
      "first_name": "Jane",
      "last_name": "Doe"
    },
    "requester": {
      "name": "John Doe",
      "first_name": "John",
      "last_name": "Doe",
      "email": "jane@example.com",
      "language": "English",
      "phone": "",
      "external_id": "",
      "field": "",
      "details": ""
    },
    "organization": {
      "name": "",
      "external_id": "",
      "details": "",
      "notes": ""
    },
    "ccs": "[]",
    "cc_names": "",
    "tags": "help-wanted",
    "current_holiday_name": "",
    "ticket_field_id": "",
    "ticket_field_option_title_id": ""
  },
  "current_user": {
    "name": "John Doe",
    "first_name": "John",
    "email": "john@example.com",
    "organization": {
      "name": "",
      "notes": "",
      "details": ""
    },
    "external_id": "",
    "phone": "",
    "details": "",
    "notes": "",
    "language": "English"
  },
  "satisfaction": {
    "current_rating": "",
    "current_comment": ""
  }
}`
