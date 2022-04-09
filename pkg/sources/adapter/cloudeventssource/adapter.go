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
)

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, aEnv adapter.EnvConfigAccessor, ceClient cloudevents.Client) adapter.Adapter {
	env := aEnv.(*envAccessor)
	logger := logging.FromContext(ctx)

	cfw, err := fs.NewCachedFileWatcher(logger)
	if err != nil {
		logger.Panicw("could not create a file watcher", zap.Error(err))
	}

	for _, as := range append(env.BasicAuths, env.Tokens...) {
		if err := cfw.Add(as.MountedValueFile); err != nil {
			logger.Panicw(
				fmt.Sprintf("authentication secret at %q could not be watched", as.MountedValueFile),
				zap.Error(err))
		}
	}

	ceh := &cloudEventsHandler{
		// corsAllowOrigin: env.CORSAllowOrigin,
		basicAuths: env.BasicAuths,
		tokens:     env.Tokens,

		cfw:      cfw,
		ceClient: ceClient,
		logger:   logging.FromContext(ctx),
	}

	// prepare CE server options

	ceServer, err := cloudevents.NewClientHTTP(
		cehttp.WithPath(env.Path),
		cehttp.WithMiddleware(ceh.handleBasicAuthentication),
		// TODO add token auth middleware
		// cehttp.WithMiddleware( /* add basic authentication */ ),

		// TODO rate liming
		// cehttp.WithRateLimiter( /* TODO */ ),
	)
	if err != nil {
		logger.Panicw("error creating CloudEvents client", zap.Error(err))
	}

	ceh.ceServer = ceServer
	return ceh
}

var _ adapter.Adapter = (*cloudEventsHandler)(nil)
