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

package cloudeventstarget

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/adapter/fs"
	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, listenClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.CloudEventsTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	if _, err := url.Parse(env.URL); err != nil {
		logger.Panicw("URL is not parseable", zap.Error(err))
	}

	fw, err := fs.NewWatcher(logger)
	if err != nil {
		logger.Panicw("Could not create a file watcher", zap.Error(err))
	}

	ceAdapter := &ceAdapter{
		listenClient: listenClient,
		logger:       logger,
		m:            sync.RWMutex{},
		sr:           metrics.MustNewEventProcessingStatsReporter(mt),
	}

	ceClientUpdater := ceAdapter.senderClientUpdater(env.URL, env.BasicAuthPasswordPath, env.BasicAuthUsername)
	if env.BasicAuthUsername != "" {

		if err := fw.Add(env.BasicAuthPasswordPath, ceClientUpdater); err != nil {
			logger.Panicw(
				"Authentication secret could not be watched at the specific path",
				zap.Error(err))
		}
		ceAdapter.fileWatcher = fw
	}

	// call the updater manually to initialize the client.
	ceClientUpdater()

	return ceAdapter
}

var _ pkgadapter.Adapter = (*ceAdapter)(nil)

type ceAdapter struct {
	fileWatcher  fs.FileWatcher
	senderClient cloudevents.Client
	listenClient cloudevents.Client

	logger *zap.SugaredLogger
	m      sync.RWMutex
	sr     *metrics.EventProcessingStatsReporter
}

func (a *ceAdapter) senderClientUpdater(url, path, username string) fs.WatchCallback {
	return func() {
		a.m.Lock()
		defer a.m.Unlock()

		opts := []cehttp.Option{
			cehttp.WithTarget(url),
		}

		if path != "" {
			opts = append(opts, cehttp.WithPath(path))
		}

		if username != "" {
			password, err := os.ReadFile(path)
			if err != nil {
				a.logger.Errorw("Could not read the mounted password at the specific path", zap.Error(err))
				return
			}

			opts = append(opts, cehttp.WithHeader(
				"Authorization",
				"Basic "+base64.StdEncoding.EncodeToString(
					append([]byte(username+":"), password...)),
			))
		}

		senderClient, err := cloudevents.NewClientHTTP(opts...)
		if err != nil {
			a.logger.Fatalw("Unable to create CloudEvent client", zap.Error(err))
		}

		a.senderClient = senderClient
	}
}

// Returns if stopCh is closed or Send() returns an error.
func (a *ceAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting CloudEvents gateway adapter")

	// If basic authentication credentials are used, start the filewatcher
	// to update the password if changed.
	if a.fileWatcher != nil {
		a.fileWatcher.Start(ctx)
	}

	return a.listenClient.StartReceiver(ctx, a.dispatch)
}

func (a *ceAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	ceTypeTag := metrics.TagEventType(event.Type())
	ceSrcTag := metrics.TagEventSource(event.Source())

	start := time.Now()
	defer func() {
		a.sr.ReportProcessingLatency(time.Since(start), ceTypeTag, ceSrcTag)
	}()

	// When using authentication sender client is initialized using the file watcher.
	// This check fails if the authentication secrets are not yet present and the
	// client has not been built.
	if a.senderClient == nil {
		err := fmt.Errorf("CloudEvents client not intialized. Please, make sure that authentication secret is available")
		a.logger.Errorw("Failed to send event", zap.Error(err))
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return nil, err
	}

	re, r := a.senderClient.Request(ctx, event)
	if cloudevents.IsNACK(r) {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		a.logger.Errorw("Could not send event to destination", zap.Error(r))
	} else {
		a.sr.ReportProcessingSuccess(ceTypeTag, ceSrcTag)
	}

	return re, r
}
