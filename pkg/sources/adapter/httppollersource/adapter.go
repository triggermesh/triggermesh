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
	"crypto/tls"
	"crypto/x509"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
)

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.HTTPPollerSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envAccessor)

	t := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: env.SkipVerify},
	}

	if env.CACertificate != "" {
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM([]byte(env.CACertificate)) {
			logger.Panicf("Failed adding certificate to pool: %s", env.CACertificate)
		}

		t.TLSClientConfig = &tls.Config{
			RootCAs: certPool,
		}
	}

	httpClient := &http.Client{Transport: t}

	httpRequest, err := http.NewRequest(env.Method, env.Endpoint, nil)
	if err != nil {
		logger.Panicw("Cannot build request", zap.Error(err))
	}

	for k, v := range env.Headers {
		httpRequest.Header.Set(k, v)
	}

	if env.BasicAuthUsername != "" || env.BasicAuthPassword != "" {
		httpRequest.SetBasicAuth(env.BasicAuthUsername, env.BasicAuthPassword)
	}

	return &httpPoller{
		eventType:   env.EventType,
		eventSource: env.EventSource,
		interval:    env.Interval,

		httpClient:  httpClient,
		httpRequest: httpRequest,

		ceClient: ceClient,
		logger:   logger,
		mt:       mt,
	}
}
