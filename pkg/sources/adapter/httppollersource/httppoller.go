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
	"context"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

type httpPoller struct {
	eventType   string
	eventSource string
	interval    time.Duration

	ceClient cloudevents.Client

	httpClient  *http.Client
	httpRequest *http.Request
	logger      *zap.SugaredLogger
	mt          *pkgadapter.MetricTag
}

var _ pkgadapter.Adapter = (*httpPoller)(nil)

// Start implements adapter.Adapter.
// Runs the server for receiving HTTP events until ctx gets cancelled.
func (h *httpPoller) Start(ctx context.Context) error {
	h.logger.Info("Starting HTTP Poller source")

	ctx = pkgadapter.ContextWithMetricTag(ctx, h.mt)

	// initial request to avoid waiting for the first tick.
	h.dispatch(ctx)

	// setup context for the request object.
	h.httpRequest = h.httpRequest.Clone(ctx)

	t := time.NewTicker(h.interval)

	for {
		select {

		case <-ctx.Done():
			h.logger.Debug("Shutting down HTTP poller")
			return nil

		case <-t.C:
			h.dispatch(ctx)
		}
	}
}

func (h *httpPoller) dispatch(ctx context.Context) {
	h.logger.Debug("Launching HTTP request")

	res, err := h.httpClient.Do(h.httpRequest)
	if err != nil {
		h.logger.Errorw("Failed sending request", zap.Error(err))
		return
	}

	defer res.Body.Close()
	resb, err := io.ReadAll(res.Body)
	if err != nil {
		h.logger.Errorw("Failed reading response body", zap.Error(err))
		return
	}

	if res.StatusCode >= 300 {
		h.logger.Errorw("Received non supported HTTP code from remote endpoint",
			zap.Int("code", res.StatusCode),
			zap.String("response", string(resb)),
		)
		return
	}

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(h.eventType)
	event.SetSource(h.eventSource)

	if err := event.SetData(cloudevents.ApplicationJSON, resb); err != nil {
		h.logger.Errorw("Failed to set event data", zap.Error(err))
		return
	}

	if result := h.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		h.logger.Errorw("Could not send Cloud Event", zap.Error(result))
	}
}
