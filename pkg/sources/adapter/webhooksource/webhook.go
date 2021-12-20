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

package webhooksource

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

const (
	serverPort                uint16 = 8080
	serverShutdownGracePeriod        = time.Second * 10
)

type webhookHandler struct {
	eventType   string
	eventSource string
	username    string
	password    string
	corsOrigin  string

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Start implements adapter.Adapter.
// Runs the server for receiving HTTP events until ctx gets cancelled.
func (h *webhookHandler) Start(ctx context.Context) error {
	m := http.NewServeMux()
	m.HandleFunc("/", h.handleAll)
	m.HandleFunc("/health", healthCheckHandler)

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", serverPort),
		Handler: m,
	}

	return runHandler(ctx, s)
}

// runHandler runs the HTTP event handler until ctx get cancelled.
func runHandler(ctx context.Context, s *http.Server) error {
	logging.FromContext(ctx).Info("Starting webhook event handler")

	errCh := make(chan error)
	go func() {
		errCh <- s.ListenAndServe()
	}()

	handleServerError := func(err error) error {
		if err != http.ErrServerClosed {
			return fmt.Errorf("during server runtime: %w", err)
		}
		return nil
	}

	select {
	case <-ctx.Done():
		logging.FromContext(ctx).Info("HTTP event handler is shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), serverShutdownGracePeriod)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			return fmt.Errorf("during server shutdown: %w", err)
		}

		return handleServerError(<-errCh)

	case err := <-errCh:
		return handleServerError(err)
	}
}

// handleAll receives all webhook events at a single resource, it
// is up to this function to parse event wrapper and dispatch.
func (h *webhookHandler) handleAll(w http.ResponseWriter, r *http.Request) {
	h.enableCors(&w)
	if r.Body == nil {
		h.handleError(errors.New("request without body not supported"), http.StatusBadRequest, w)
		return
	}

	if h.username != "" && h.password != "" {
		us, ps, ok := r.BasicAuth()
		if !ok {
			h.handleError(errors.New("wrong authentication header"), http.StatusBadRequest, w)
			return
		}
		if us != h.username || ps != h.password {
			h.handleError(errors.New("credentials are not valid"), http.StatusUnauthorized, w)
			return
		}
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.handleError(err, http.StatusInternalServerError, w)
		return
	}

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(h.eventType)
	event.SetSource(h.eventSource)

	if err := event.SetData(r.Header.Get("Content-Type"), body); err != nil {
		h.handleError(fmt.Errorf("failed to set event data: %w", err), http.StatusInternalServerError, w)
		return
	}

	if result := h.ceClient.Send(context.Background(), event); !cloudevents.IsACK(result) {
		h.handleError(fmt.Errorf("could not send Cloud Event: %w", result), http.StatusInternalServerError, w)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *webhookHandler) handleError(err error, code int, w http.ResponseWriter) {
	h.logger.Error("An error ocurred", zap.Error(err))
	http.Error(w, err.Error(), code)
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (h *webhookHandler) enableCors(w *http.ResponseWriter) {
	if h.corsOrigin != "" {
		(*w).Header().Set("Access-Control-Allow-Origin", h.corsOrigin)
	}
}
