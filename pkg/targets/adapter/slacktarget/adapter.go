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

package slacktarget

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	"github.com/triggermesh/triggermesh/pkg/targets/adapter/slacktarget/slack"
)

const (
	apiURL = "https://slack.com/api/"
	// method URL will be obtained removing this prefix from
	// the received event type.
	eventTypePrefix = "com.slack.webapi."
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.SlackTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	// catalog is the supported operations registry, which can also
	// be used in the future to filter available methods from the
	// adapter. At this moment we use the full available cataglo and
	// let users filter by configuring the bot at slack with the
	// set of OAuth scopes that fit the users needs.
	catalog := slack.GetFullCatalog(true)

	return &slackAdapter{
		slackClient: slack.NewWebAPIClient(env.Token, apiURL, &http.Client{}, catalog),
		ceClient:    ceClient,
		logger:      logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*slackAdapter)(nil)

type slackAdapter struct {
	slackClient slack.WebAPIClient

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (t *slackAdapter) Start(ctx context.Context) error {
	t.logger.Info("Starting Slack adapter")

	if err := t.ceClient.StartReceiver(ctx, t.dispatch); err != nil {
		return err
	}
	return nil
}

func (t *slackAdapter) dispatch(event cloudevents.Event) cloudevents.Result {
	// Take a cloud event as passed in, and submit a message

	et := event.Type()
	if !strings.HasPrefix(et, eventTypePrefix) {
		t.logger.Errorw("Unsupported event type", zap.String("error", "event type is not supported: "+et))
		return cloudevents.ResultNACK
	}

	methodURL := et[len(eventTypePrefix):]
	res, err := t.slackClient.Do(methodURL, event.Data())
	if err != nil {
		t.logger.Errorw("Unable to send message", zap.Error(err))
		return cloudevents.ResultNACK
	}

	if res.Warning() != "" {
		t.logger.Warn(res.Warning())
	}

	if !res.IsOK() {
		t.logger.Errorw("Request failed", zap.String("error", res.Error()))
		return cloudevents.ResultNACK
	}

	// TODO return event containing response structure
	// See: https://github.com/triggermesh/knative-targets/issues/165

	return cloudevents.ResultACK
}
