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

package ocimetricssource

import (
	"context"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

var _ pkgadapter.Adapter = (*ociMetricsAdapter)(nil)

type ociMetricsAdapter struct {
	handler OCIMetricsAPIHandler
	logger  *zap.SugaredLogger
	mt      *pkgadapter.MetricTag
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.OCIMetricsSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envAccessor)

	return &ociMetricsAdapter{
		handler: NewOCIMetricsAPIHandler(ceClient, env, v1alpha1.OCIGenerateEventSource(env.Namespace, env.Name), logger.Named("handler")),
		logger:  logger,
		mt:      mt,
	}

}

// Start implements adapter.Adapter.
func (o *ociMetricsAdapter) Start(ctx context.Context) error {
	ctx = pkgadapter.ContextWithMetricTag(ctx, o.mt)
	return o.handler.Start(ctx)
}
