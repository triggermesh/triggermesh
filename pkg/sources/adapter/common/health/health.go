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

// Package health contains helpers to enable HTTP health checking.
package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"knative.dev/pkg/logging"
)

const healthPath = "/health"

// Use a var instead of a const to allow tests to override this value.
var healthPort uint16 = 8080

const gracefulHandlerShutdown = 3 * time.Second

// handler serves requests to the health endpoint. It returns a success HTTP
// code when its value is true.
type handler struct {
	sync.RWMutex
	ready bool
}

// Verify that handler implements http.Handler.
var _ http.Handler = (*handler)(nil)

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if !h.isReady() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) isReady() bool {
	h.RLock()
	defer h.RUnlock()

	return h.ready
}

var defaultHandler handler

// Start runs the default HTTP health handler.
func Start(ctx context.Context) {
	mux := &http.ServeMux{}
	mux.Handle(healthPath, &defaultHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", healthPort),
		Handler: mux,
	}

	errCh := make(chan error)

	go func() {
		errCh <- server.ListenAndServe()
	}()

	handleServerError := func(err error) {
		if err != http.ErrServerClosed {
			logging.FromContext(ctx).Errorw("Error during runtime of health server", zap.Error(err))
		}
	}

	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), gracefulHandlerShutdown)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logging.FromContext(ctx).Errorw("Error during shutdown of health server", zap.Error(err))
		}

		handleServerError(<-errCh)

	case err := <-errCh:
		handleServerError(err)
	}
}

// MarkReady indicates that the application is ready to operate.
func MarkReady() {
	if defaultHandler.isReady() {
		return
	}

	defaultHandler.Lock()
	defer defaultHandler.Unlock()

	// double-checked lock to ensure we don't write the value of "ready"
	// twice if multiple goroutines called MarkReady() simultaneously.
	if defaultHandler.ready {
		return
	}

	defaultHandler.ready = true
}
