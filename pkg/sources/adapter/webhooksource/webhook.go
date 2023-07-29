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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"
)

const (
	serverPort                uint16 = 8080
	serverShutdownGracePeriod        = time.Second * 10
	queryPrefix                      = "q"
	headerPrefix                     = "h"
)

type webhookHandler struct {
	eventType               string
	eventSource             string
	extensionAttributesFrom *ExtensionAttributesFrom
	username                string
	password                string
	corsAllowOrigin         string

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
	mt       *pkgadapter.MetricTag
}

// Start implements pkgadapter.Adapter
// Runs the server for receiving HTTP events until ctx gets cancelled.
func (h *webhookHandler) Start(ctx context.Context) error {
	ctx = pkgadapter.ContextWithMetricTag(ctx, h.mt)

	m := http.NewServeMux()
	m.HandleFunc("/", h.handleAll(ctx))
	m.HandleFunc("/health", healthCheckHandler)

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", serverPort),
		Handler: m,
	}

	return runHandler(ctx, s)
}

// runHandler runs the HTTP event handler until ctx gets cancelled.
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
func (h *webhookHandler) handleAll(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.corsAllowOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", h.corsAllowOrigin)
		}

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
		body, err := io.ReadAll(r.Body)
		if err != nil {
			h.handleError(err, http.StatusInternalServerError, w)
			return
		}

		event := cloudevents.NewEvent(cloudevents.VersionV1)
		event.SetType(h.eventType)
		event.SetSource(h.eventSource)

		// Add extension attributes if configured
		if h.extensionAttributesFrom != nil {
			if h.extensionAttributesFrom.path {
				event.SetExtension("path", r.URL.Path)
			}
			if h.extensionAttributesFrom.method {
				event.SetExtension("method", r.Method)
			}
			if h.extensionAttributesFrom.host {
				event.SetExtension("host", r.Host)
			}
			if h.extensionAttributesFrom.queries {
				for k, v := range r.URL.Query() {
					if len(v) == 1 {
						event.SetExtension(sanitizeCloudEventAttributeName(queryPrefix+k), v[0])
					} else {
						for i := range v {
							event.SetExtension(sanitizeCloudEventAttributeName(
								fmt.Sprintf("%s%s%d", queryPrefix, k, i)), v[i])
						}
					}
				}
			}
			if h.extensionAttributesFrom.headers {
				for k, v := range r.Header {
					// Prevent Authorization header from being added
					// as a CloudEvent attribute
					if k == "Authorization" {
						continue
					}
					if k == "Ce-Id" {
						if len(v) != 0 {
							event.SetID(v[0])
						}
						continue
					}
					if k == "Ce-Subject" {
						if len(v) != 0 {
							event.SetSubject(v[0])
						}
						continue
					}

					if len(v) == 1 {
						event.SetExtension(sanitizeCloudEventAttributeName(headerPrefix+k), v[0])
					} else {
						for i := range v {
							event.SetExtension(sanitizeCloudEventAttributeName(
								fmt.Sprintf("%s%s%d", headerPrefix, k, i)), v[i])
						}
					}
				}
			}
		}

		if err := event.SetData(r.Header.Get("Content-Type"), body); err != nil {
			h.handleError(fmt.Errorf("failed to set event data: %w", err), http.StatusInternalServerError, w)
			return
		}

		rEvent, result := h.ceClient.Request(ctx, event)
		if !cloudevents.IsACK(result) {
			h.handleError(fmt.Errorf("could not send Cloud Event: %w", result), http.StatusInternalServerError, w)
			return
		}
		if rEvent == nil || rEvent.Data() == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (h *webhookHandler) handleError(err error, code int, w http.ResponseWriter) {
	h.logger.Errorw("An error ocurred", zap.Error(err))
	http.Error(w, err.Error(), code)
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func sanitizeCloudEventAttributeName(name string) string {
	// only lowercase accepted
	name = strings.ToLower(name)

	// strip non valid characters
	needsStripping := false
	for i := range name {
		if !((name[i] >= 'a' && name[i] <= 'z') || (name[i] >= '0' && name[i] <= '9')) {
			needsStripping = true
			break
		}
	}

	if needsStripping {
		stripped := []byte{}
		for i := range name {
			if (name[i] >= 'a' && name[i] <= 'z') || (name[i] >= '0' && name[i] <= '9') {
				stripped = append(stripped, name[i])
			}
		}
		name = string(stripped)
	}

	// truncate if longer than 20 characters
	if len(name) > 20 {
		name = name[:20]
	}

	// data is a reserved element at CloudEvents
	if name == "data" || name == "path" || name == "method" || name == "host" {
		return "data0"
	}
	return name
}
