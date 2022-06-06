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

package azureiothubsource

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/amenzhinsky/iothub/iotservice"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig
	ConnectionString string `envconfig:"IOTHUB_CONNECTION_STRING" required:"true"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger   *zap.SugaredLogger
	mt       *pkgadapter.MetricTag
	ceClient cloudevents.Client
	source   string
	c        *iotservice.Client
}

var _ pkgadapter.Adapter = (*adapter)(nil)

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.AzureIOTHubSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	s := strings.Split(env.ConnectionString, ";")
	endpoint := s[0]
	sbURL := strings.Split(endpoint, "=")

	c, err := iotservice.NewFromConnectionString(
		env.ConnectionString,
	)
	if err != nil {
		logger.Fatalw("Failed to obtain IoT client", zap.Error(err))
	}

	return &adapter{
		logger:   logger,
		mt:       mt,
		ceClient: ceClient,
		c:        c,
		source:   sbURL[1],
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Azure IoT Hub Source adapter")

	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	return a.c.SubscribeEvents(ctx, a.eventHandler(ctx))
}

func (a *adapter) eventHandler(ctx context.Context) iotservice.EventHandler {
	return func(msg *iotservice.Event) error {
		event := cloudevents.NewEvent(cloudevents.VersionV1)
		event.SetType(v1alpha1.AzureEventType(sources.AzureIOTHub, v1alpha1.AzureIOTHubGenericEventType))
		event.SetSource(a.source)
		if err := event.SetData(cloudevents.ApplicationJSON, msg); err != nil {
			return fmt.Errorf("setting event data: %w", err)
		}

		if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
			return fmt.Errorf("sending CloudEvent: %w", result)
		}

		return nil
	}
}
