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

package twiliosource

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

const (
	serverPort                uint16 = 8080
	serverShutdownGracePeriod        = time.Second * 10
)

// adapter implements the source's adapter.
type adapter struct {
	ceClient    cloudevents.Client
	eventsource string
	logger      *zap.SugaredLogger
	mt          *pkgadapter.MetricTag
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &pkgadapter.EnvConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.TwilioSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	return &adapter{
		ceClient:    ceClient,
		eventsource: v1alpha1.TwilioSourceName(envAcc.GetNamespace(), envAcc.GetName()),
		logger:      logging.FromContext(ctx),
		mt:          mt,
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

// Start implements adapter.Adapter.
// Runs the server for receiving HTTP events until ctx gets cancelled.
func (h *adapter) Start(ctx context.Context) error {
	ctx = pkgadapter.ContextWithMetricTag(ctx, h.mt)

	m := http.NewServeMux()
	m.HandleFunc("/", h.handleRoot(ctx))
	m.HandleFunc("/health", healthCheckHandler)

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", serverPort),
		Handler: m,
	}

	return runHandler(ctx, s)
}

// runHandler runs the HTTP event handler until ctx get cancelled.
func runHandler(ctx context.Context, s *http.Server) error {
	logging.FromContext(ctx).Info("Starting HTTP event handler")

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

func (h *adapter) handleRoot(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		h.logger.Debug("Got request: ", *req)

		m := parseFormToMessage(req)

		if err := h.sendCloudEvent(ctx, m); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (h *adapter) sendCloudEvent(ctx context.Context, m *Message) error {
	h.logger.Debug("Sending CloudEvent")

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(v1alpha1.TwilioSourceGenericEventType)
	event.SetSource(h.eventsource)
	if err := event.SetData(cloudevents.ApplicationJSON, m); err != nil {
		return fmt.Errorf("failed to set event data: %w", err)
	}

	if result := h.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		return fmt.Errorf("failed to send CloudEvent: %w", result)
	}
	return nil
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func parseFormToMessage(req *http.Request) *Message {
	m := &Message{}

	m.MessageSid = req.FormValue("MessageSid")
	m.From = req.FormValue("From")
	m.Body = req.FormValue("Body")
	m.To = req.FormValue("To")
	m.SmsMessageSid = req.FormValue("SmsMessageSid")
	m.SmsStatus = req.FormValue("SmsStatus")
	m.FromCountry = req.FormValue("FromCountry")
	m.NumSegments = req.FormValue("NumSegments")
	m.ToZip = req.FormValue("ToZip")
	m.NumMeda = req.FormValue("NumMeda")
	m.AccountSid = req.FormValue("AccountSid")
	m.APIVersion = req.FormValue("ApiVersion")
	m.ToCountry = req.FormValue("ToCountry")
	m.ToCity = req.FormValue("ToCity")
	m.FromZip = req.FormValue("FromZip")
	m.SmsSid = req.FormValue("SmsSid")
	m.FromState = req.FormValue("FromState")
	m.FromCity = req.FormValue("FromCity")
	m.ToState = req.FormValue("ToState")

	return m
}
