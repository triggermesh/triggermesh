/*
Copyright 2020 TriggerMesh Inc.

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

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"go.uber.org/zap"
	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"
)

var _ adapter.Adapter = (*ociMetricsAdapter)(nil)

type ociMetricsAdapter struct {
	handler OCIMetricsAPIHandler
	logger  *zap.SugaredLogger
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, aEnv adapter.EnvConfigAccessor, ceClient cloudevents.Client) adapter.Adapter {
	env := aEnv.(*envAccessor)
	logger := logging.FromContext(ctx)

	return &ociMetricsAdapter{
		handler: NewOCIMetricsAPIHandler(ceClient, aEnv, v1alpha1.OCIGenerateEventSource(env.Namespace, env.Name), logger.Named("handler")),
		logger:  logger,
	}

}

// Start implements adapter.Adapter.
func (o *ociMetricsAdapter) Start(ctx context.Context) error {
	return o.handler.Start(ctx)
}
