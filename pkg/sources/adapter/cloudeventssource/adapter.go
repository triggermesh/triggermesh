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

package cloudeventssource

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"
)

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, aEnv adapter.EnvConfigAccessor, ceClient cloudevents.Client) adapter.Adapter {
	env := aEnv.(*envAccessor)
	logger := logging.FromContext(ctx)

	ceServer, err := cloudevents.NewClientHTTP(
	// TODO rate liming
	// cehttp.WithRateLimiter( /* TODO */ ),

	// TODO add basic auth middleware
	// cehttp.WithMiddleware(/* add basic authentication */),

	// TODO add token middleware
	// cehttp.WithMiddleware(/* add basic authentication */),
	)

	logger.Infof("DEBUG DELETEME RateLimiter %v", env.RateLimiterRPS)
	logger.Infof("DEBUG DELETEME BasicAuths %v", env.BasicAuths)
	logger.Infof("DEBUG DELETEME Tokens %v", env.Tokens)

	if err != nil {
		logger.Panicw("error creating CloudEvents client", zap.Error(err))
	}

	return &cloudEventsHandler{
		basicAuths: env.BasicAuths,
		tokens:     env.Tokens,
		// corsAllowOrigin: env.CORSAllowOrigin,

		ceClient: ceClient,
		ceServer: ceServer,
		logger:   logging.FromContext(ctx),
	}
}

var _ adapter.Adapter = (*cloudEventsHandler)(nil)
