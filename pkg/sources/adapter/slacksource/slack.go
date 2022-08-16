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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

const apiAppIDCeExtension = "comslackapiappid"

// SlackEventAPIHandler listen for Slack API Events
type SlackEventAPIHandler interface {
	Start(ctx context.Context) error
}

type slackEventAPIHandler struct {
	port          int
	signingSecret string
	appID         string

	ceClient cloudevents.Client
	srv      *http.Server

	time   timeWrap
	logger *zap.SugaredLogger
}

// NewSlackEventAPIHandler creates the default implementation of the Slack API Events handler
func NewSlackEventAPIHandler(ceClient cloudevents.Client, port int, signingSecret, appID string, tw timeWrap, logger *zap.SugaredLogger) SlackEventAPIHandler {
	return &slackEventAPIHandler{
		port:          port,
		signingSecret: signingSecret,
		appID:         appID,

		ceClient: ceClient,
		time:     tw,
		logger:   logger,
	}
}

// Start the server for receiving Slack callbacks. Will block
// until the stop channel closes.
func (h *slackEventAPIHandler) Start(ctx context.Context) error {
	h.logger.Info("Starting Slack event handler")

	m := http.NewServeMux()
	m.HandleFunc("/", h.handleAll(ctx))

	h.srv = &http.Server{
		Addr:    ":" + strconv.Itoa(h.port),
		Handler: m,
	}

	done := make(chan bool, 1)
	go h.gracefulShutdown(ctx.Done(), done)

	h.logger.Debugf("Server is ready to handle requests at %s", h.srv.Addr)
	if err := h.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("could not listen on %s: %w", h.srv.Addr, err)
	}

	<-done
	h.logger.Debug("Server stopped")
	return nil
}

// handleAll receives all Slack events at a single resource, it
// is up to this function to parse event wrapper and dispatch.
func (h *slackEventAPIHandler) handleAll(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			h.handleError(errors.New("request without body not supported"), http.StatusBadRequest, w)
			return
		}

		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			h.handleError(err, http.StatusInternalServerError, w)
			return
		}

		if h.signingSecret != "" {
			err = h.verifySigning(r.Header, body)
			if err != nil {
				h.handleError(err, http.StatusUnauthorized, w)
				return
			}
		}

		event := &SlackEventWrapper{}
		err = json.Unmarshal(body, event)
		if err != nil {
			h.handleError(fmt.Errorf("could not unmarshal JSON request: %w", err), http.StatusBadRequest, w)
			return
		}

		// There are only 2 documented types to be received from the Events API
		// - `event_callback`, See: https://api.slack.com/events-api#receiving_events
		// - `url_verification`, See: https://api.slack.com/events-api#subscriptions
		switch eventType := sanitizeUserInput(event.Type); eventType {
		case "event_callback":
			// All paths that are not managed by this integration and are
			// not errors need to return 2xx withing 3 seconds to Slack API.
			// Otherwise the message will be retried.
			// See: https://api.slack.com/events-api#receiving_events (Responding to Events)
			if h.appID != "" && event.APIAppID != h.appID {
				return
			}

			h.handleCallback(ctx, event, w)

		case "url_verification":
			// url_verification does not include an appID so there is no way to
			// filter against it

			h.handleChallenge(body, w)

		default:
			h.logger.Warn("Content not supported: ", strconv.Quote(eventType))
		}
	}
}

func (h *slackEventAPIHandler) gracefulShutdown(stopCh <-chan struct{}, done chan<- bool) {
	<-stopCh
	h.logger.Debug("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h.srv.SetKeepAlivesEnabled(false)
	if err := h.srv.Shutdown(ctx); err != nil {
		h.logger.Fatalf("Could not gracefully shutdown the server: %v", err)
	}
	close(done)
}

func (h *slackEventAPIHandler) handleError(err error, code int, w http.ResponseWriter) {
	h.logger.Errorw("An error ocurred", zap.Error(err))
	http.Error(w, err.Error(), code)
}

func (h *slackEventAPIHandler) handleChallenge(body []byte, w http.ResponseWriter) {
	h.logger.Debug("Challenge received")
	c := &SlackChallenge{}

	err := json.Unmarshal(body, c)
	if err != nil {
		h.handleError(err, http.StatusBadRequest, w)
		return
	}

	cr := &SlackChallengeResponse{Challenge: c.Challenge}
	res, err := json.Marshal(cr)
	if err != nil {
		h.handleError(err, http.StatusBadRequest, w)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(res)
	if err != nil {
		h.handleError(err, http.StatusInternalServerError, w)
	}
}

func (h *slackEventAPIHandler) handleCallback(ctx context.Context, wrapper *SlackEventWrapper, w http.ResponseWriter) {
	h.logger.Debug("Callback received")

	event, err := cloudEventFromEventWrapper(wrapper)
	if err != nil {
		h.handleError(err, http.StatusBadRequest, w)
		return
	}

	if result := h.ceClient.Send(ctx, *event); !cloudevents.IsACK(result) {
		h.handleError(err, http.StatusInternalServerError, w)
	}
}

func cloudEventFromEventWrapper(wrapper *SlackEventWrapper) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent(cloudevents.VersionV1)

	event.SetID(wrapper.EventID)
	event.SetType(v1alpha1.SlackGenericEventType)
	event.SetSource(wrapper.TeamID)
	event.SetExtension(apiAppIDCeExtension, wrapper.APIAppID)
	event.SetTime(time.Unix(int64(wrapper.EventTime), 0))
	event.SetSubject(wrapper.Event.Type())
	if err := event.SetData(cloudevents.ApplicationJSON, wrapper.Event); err != nil {
		return nil, err
	}

	return &event, nil
}

var newlineToSpace = strings.NewReplacer("\n", " ", "\r", " ")

// sanitizeUserInput removes unwanted characters from the given string.
// It also guarantees the safe logging of data that potentially originates from
// user input (CWE-117, https://cwe.mitre.org/data/definitions/117.html).
func sanitizeUserInput(s string) string {
	return newlineToSpace.Replace(s)
}
