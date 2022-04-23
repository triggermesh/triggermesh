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

package cloudeventssource

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"go.uber.org/zap"

	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/adapter/fs"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/cloudeventssource/ratelimiter"
)

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, aEnv adapter.EnvConfigAccessor, ceClient cloudevents.Client) adapter.Adapter {
	env := aEnv.(*envAccessor)
	logger := logging.FromContext(ctx)

	cfw, err := fs.NewCachedFileWatcher(logger)
	if err != nil {
		logger.Panicw("Could not create a file watcher", zap.Error(err))
	}

	for _, as := range env.BasicAuths {
		if err := cfw.Add(as.MountedValueFile); err != nil {
			logger.Panicw(
				fmt.Sprintf("Authentication secret at %q could not be watched", as.MountedValueFile),
				zap.Error(err))
		}
	}

	ceh := &cloudEventsHandler{
		basicAuths: env.BasicAuths,

		cfw:      cfw,
		ceClient: ceClient,
		logger:   logging.FromContext(ctx),
	}

	// prepare CE server options
	options := []cehttp.Option{}

	if env.Path != "" {
		options = append(options, cehttp.WithPath(env.Path))
	}
	if len(env.BasicAuths) != 0 {
		options = append(options, cehttp.WithMiddleware(ceh.handleAuthentication))
	}

	if env.RequestsPerSecond != 0 {
		rl, err := ratelimiter.New(env.RequestsPerSecond)
		if err != nil {
			logger.Panicw("Could not create rate limiter", zap.Error(err))
		}
		options = append(options, cehttp.WithRateLimiter(rl))
	}

	ceServer, err := cloudevents.NewClientHTTP(options...)
	if err != nil {
		logger.Panicw("Error creating CloudEvents client", zap.Error(err))
	}

	ceh.ceServer = ceServer
	return ceh
}

var _ adapter.Adapter = (*cloudEventsHandler)(nil)
