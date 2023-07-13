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

package googlecloudpubsubtarget

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)
	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.GoogleCloudPubSubResponseEventType),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.GoogleCloudPubSubTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	opts := make([]option.ClientOption, 0)
	if env.ServiceAccountKey != nil {
		opts = append(opts, option.WithCredentialsJSON(env.ServiceAccountKey))
	}

	psCli, err := pubsub.NewClient(ctx, env.TopicName.Project, opts...)
	if err != nil {
		logger.Panicw("Failed to create Google Cloud Pub/Sub API client", zap.Error(err))
	}

	t := psCli.Topic(env.TopicName.Resource)
	return &adapter{
		topic: t,

		replier:          replier,
		ceClient:         ceClient,
		logger:           logger,
		mt:               mt,
		sr:               metrics.MustNewEventProcessingStatsReporter(mt),
		discardCEContext: env.DiscardCEContext,
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

type adapter struct {
	topic *pubsub.Topic

	replier          *targetce.Replier
	ceClient         cloudevents.Client
	logger           *zap.SugaredLogger
	mt               *pkgadapter.MetricTag
	sr               *metrics.EventProcessingStatsReporter
	discardCEContext bool
}

func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting GoogleCloudPubSubTarget Adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *adapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	ceTypeTag := metrics.TagEventType(event.Type())
	ceSrcTag := metrics.TagEventSource(event.Source())
	start := time.Now()
	defer func() {
		a.sr.ReportProcessingLatency(time.Since(start), ceTypeTag, ceSrcTag)
	}()

	var data []byte

	if a.discardCEContext {
		data = event.Data()
	} else {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
		data = jsonEvent
	}

	result := a.topic.Publish(ctx, &pubsub.Message{
		Data: data,
	})
	id, err := result.Get(ctx)
	if err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	a.logger.Debugf("Published a message; msg ID: %v\n", id)
	a.sr.ReportProcessingSuccess(ceTypeTag, ceSrcTag)
	return a.replier.Ok(&event, "ok")
}
