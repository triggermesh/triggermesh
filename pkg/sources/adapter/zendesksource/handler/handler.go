/*
Copyright 2021 TriggerMesh Inc.

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

// Package handler contains the logic for handling Zendesk webhook events.
package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	"knative.dev/pkg/logging/logkey"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

const (
	headerAuthKey    = "Authorization"
	headerAuthPrefix = "Basic " // the trailing space must not be removed

	headerContentTypeKey  = "Content-Type"
	headerContentTypeJSON = "application/json"

	ceExtTicketType = "tickettype"
)

// Handler handles the events sent by a Zendesk trigger.
type Handler struct {
	logger *zap.SugaredLogger

	ceClient cloudevents.Client
	eventSrc string
	sink     string

	// base64 encoded username:password
	base64UsrPass string
}

// Check that Handler implements http.Handler.
var _ http.Handler = (*Handler)(nil)

// New returns an initialized Handler.
func New(src *v1alpha1.ZendeskSource, logger *zap.SugaredLogger, ceClient cloudevents.Client,
	username, password string) *Handler {

	return &Handler{
		logger: logger.With(zap.String(logkey.Key, src.Namespace+"/"+src.Name)),

		ceClient: ceClient,
		eventSrc: src.AsEventSource(),
		sink:     src.Status.SinkURI.String(),

		base64UsrPass: base64.StdEncoding.EncodeToString([]byte(username + ":" + password)),
	}
}

// ServeHTTP implements http.Handler.
// Converts incoming Zesndesk events to CloudEvents and sends them to the configured event sink.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Unsupported method "+r.Method, http.StatusMethodNotAllowed)
		return
	}

	// handle semicolon-separated header value (e.g. "application/json; charset=utf-8")
	ctHeader := r.Header.Get(headerContentTypeKey)
	if strings.TrimSpace(strings.SplitN(ctHeader, ";", 2)[0]) != headerContentTypeJSON {
		http.Error(w, "Unsupported media type "+html.EscapeString(ctHeader), http.StatusUnsupportedMediaType)
		return
	}

	if err := validateAuthHeader(r, h.base64UsrPass); err != nil {
		handleError("Failed to validate auth header", err, http.StatusUnauthorized, h.logger, w)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handleError("Failed to read request body", err, http.StatusInternalServerError, h.logger, w)
		return
	}

	data := &ticketCreated{}
	if err := json.Unmarshal(body, data); err != nil {
		handleError("Failed to parse event data", err, http.StatusBadRequest, h.logger, w)
		return
	}

	event := cloudevents.NewEvent(cloudevents.VersionV1)

	event.SetType(v1alpha1.ZendeskTicketCreatedEventType)
	event.SetSource(h.eventSrc)
	event.SetSubject(strconv.Itoa(data.Ticket.ID))

	if ticketType := data.Ticket.Type; ticketType != "" {
		event.SetExtension(ceExtTicketType, ticketType)
	}

	if err := event.SetData(cloudevents.ApplicationJSON, json.RawMessage(body)); err != nil {
		handleError("Failed to set event data", err, http.StatusInternalServerError, h.logger, w)
		return
	}

	ctx := cloudevents.ContextWithTarget(context.Background(), h.sink)

	if result := h.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		handleError("Failed to send CloudEvent", err, http.StatusInternalServerError, h.logger, w)
	}
}

// validateAuthHeader verifies that the request contains a valid Basic Auth header.
// NOTE(antoineco): do not use Zendesk's "Test Target" action to troubleshoot a
// Target, it always sends a blank password.
func validateAuthHeader(r *http.Request, expectVal string) error {
	auth := r.Header.Get(headerAuthKey)
	if !strings.HasPrefix(auth, headerAuthPrefix) {
		return errors.New("incorrect header prefix")
	}

	if auth[len(headerAuthPrefix):] != expectVal {
		return errors.New("invalid credentials")
	}

	return nil
}

// handleError logs the given error and writes it to the given ResponseWriter.
func handleError(msg string, err error, httpCode int, logger *zap.SugaredLogger, w http.ResponseWriter) {
	logger.Errorw(msg, zap.Error(err))
	http.Error(w, fmt.Sprint(msg, ": ", err), httpCode)
}
