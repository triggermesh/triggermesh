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

package xslt

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/common/router"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/eventing/pkg/kncloudevents"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
)

const serverPort int = 8080

// adapter implements the component's adapter.
type adapter struct {
	logger *zap.SugaredLogger

	// receiver receives incoming HTTP requests
	receiver *kncloudevents.HTTPMessageReceiver

	// fields accessed during object reconciliation
	router *router.Router
}

// NewAdapter returns a constructor for the source's adapter.
func NewAdapter(component string) pkgadapter.AdapterConstructor {
	return func(ctx context.Context, _ pkgadapter.EnvConfigAccessor,
		_ cloudevents.Client) pkgadapter.Adapter {

		return &adapter{
			logger: logging.FromContext(ctx),

			receiver: kncloudevents.NewHTTPMessageReceiver(serverPort),

			router: &router.Router{},
		}
	}
}

// Start begins to receive messages for the handler.
//
// HTTP POST requests to the root path (/) are accepted.
//
// This method will block until ctx is done.
func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting XML Target adapter")
	return a.receiver.StartListen(ctx, a)
}

func (a *adapter) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Allow", "POST, OPTIONS")

	if request.Method == http.MethodOptions {
		writer.Header().Set("WebHook-Allowed-Origin", "*") // Accept from any Origin
		writer.Header().Set("WebHook-Allowed-Rate", "*")   // Unlimited requests/minute
		writer.WriteHeader(http.StatusOK)
		return
	}

	if request.Method != http.MethodPost {
		a.logger.Warn("unexpected request method", zap.String("method", request.Method))
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	a.logger.Warn("received call, TODO")
}

// RegisterHandlerFor implements MTAdapter.
func (a *adapter) RegisterHandlerFor(ctx context.Context, src *v1alpha1.XsltTransform) error {
	a.logger.Warn("registering instance, TODO")
	return nil
}

// RegisterHandlerFor implements MTAdapter.
func (a *adapter) DeregisterHandlerFor(ctx context.Context, src *v1alpha1.XsltTransform) error {
	a.logger.Warn("deregistering instance, TODO")
	return nil
}
