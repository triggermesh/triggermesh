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

package awssnssource

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/apis"
	k8sclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/client/generated/injection/client"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/awssnssource/handler"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/awssnssource/status"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/env"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/router"
	snsclient "github.com/triggermesh/triggermesh/pkg/sources/client/sns"
	"github.com/triggermesh/triggermesh/pkg/sources/routing"
)

// adapter implements the source's adapter.
type adapter struct {
	logger *zap.SugaredLogger

	ceClient cloudevents.Client
	snsCg    snsclient.ClientGetter

	// fields accessed during object reconciliation
	router        *router.Router
	statusPatcher *status.Patcher
}

// Check the interfaces adapter should implement.
var (
	_ pkgadapter.Adapter = (*adapter)(nil)
	_ MTAdapter          = (*adapter)(nil)
	_ http.Handler       = (*adapter)(nil)
)

// NewEnvConfig satisfies env.ConfigConstructor.
func NewEnvConfig() env.ConfigAccessor {
	return &env.Config{}
}

// NewAdapter returns a constructor for the source's adapter.
func NewAdapter(component string) pkgadapter.AdapterConstructor {
	return func(ctx context.Context, _ pkgadapter.EnvConfigAccessor,
		ceClient cloudevents.Client) pkgadapter.Adapter {

		ns := injection.GetNamespaceScope(ctx)
		secrGetter := secretGetter(k8sclient.Get(ctx).CoreV1().Secrets(ns))
		srcClient := client.Get(ctx).SourcesV1alpha1().AWSSNSSources(ns)

		return &adapter{
			logger: logging.FromContext(ctx),

			ceClient: ceClient,
			snsCg:    snsclient.NewClientGetter(secrGetter),

			router:        &router.Router{},
			statusPatcher: status.NewPatcher(component, srcClient),
		}
	}
}

func secretGetter(cli coreclientv1.SecretInterface) snsclient.NamespacedSecretsGetter {
	return func(string) coreclientv1.SecretInterface {
		return cli
	}
}

const (
	serverPort                uint16 = 8080
	serverShutdownGracePeriod        = time.Second * 10
)

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    fmt.Sprint(":", serverPort),
		Handler: a,
	}

	return runHandler(ctx, server)
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

// ServeHTTP implements http.Handler.
// Delegates incoming requests to the underlying router.
func (a *adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// RegisterHandlerFor implements MTAdapter.
func (a *adapter) RegisterHandlerFor(ctx context.Context, src *v1alpha1.AWSSNSSource) error {
	snsCli, err := a.snsCg.Get(src)
	if err != nil {
		return fmt.Errorf("obtaining SNS client: %w", err)
	}

	h := handler.New(src, a.logger, a.ceClient, snsCli)

	a.router.RegisterPath(routing.URLPath(src), h)
	return nil
}

// DeregisterHandlerFor implements MTAdapter.
func (a *adapter) DeregisterHandlerFor(ctx context.Context, src *v1alpha1.AWSSNSSource) error {
	a.router.DeregisterPath(routing.URLPath(src))
	return nil
}

// PropagateCondition implements MTAdapter.
func (a *adapter) PropagateCondition(ctx context.Context, src *v1alpha1.AWSSNSSource, cond *apis.Condition) error {
	return status.PropagateCondition(ctx, a.statusPatcher, src, cond)
}
