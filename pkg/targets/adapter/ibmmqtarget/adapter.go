//go:build !noclibs

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

package ibmmqtarget

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
	"github.com/triggermesh/triggermesh/pkg/targets/adapter/ibmmqtarget/mq"
)

var _ pkgadapter.Adapter = (*ibmmqtargetAdapter)(nil)

type ibmmqtargetAdapter struct {
	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
	mqEnvs   *TargetEnvAccessor
	queue    *mq.Object

	sr *metrics.EventProcessingStatsReporter
}

// NewAdapter adapter implementation
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.IBMMQTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*TargetEnvAccessor)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.IBMMQTargetGenericResponseEventType),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &ibmmqtargetAdapter{
		replier:  replier,
		ceClient: ceClient,
		logger:   logger,
		mqEnvs:   env,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

// Returns if stopCh is closed or Send() returns an error.
func (a *ibmmqtargetAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting IBMMQTarget Adapter")

	conn, err := mq.NewConnection(a.mqEnvs.ConnectionConfig, a.mqEnvs.Auth)
	if err != nil {
		return fmt.Errorf("failed to create IBM MQ connection: %w", err)
	}
	defer conn.Disc()

	queue, err := mq.OpenQueue(a.mqEnvs.QueueName, &a.mqEnvs.ReplyTo, conn)
	if err != nil {
		return fmt.Errorf("failed to open IBM MQ queue: %w", err)
	}
	defer queue.Close()

	a.queue = &queue

	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *ibmmqtargetAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	ceTypeTag := metrics.TagEventType(event.Type())
	ceSrcTag := metrics.TagEventSource(event.Source())

	start := time.Now()
	defer func() {
		a.sr.ReportProcessingLatency(time.Since(start), ceTypeTag, ceSrcTag)
	}()

	var msg []byte

	if a.mqEnvs.DiscardCEContext {
		msg = event.Data()
	} else {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}
		msg = jsonEvent
	}

	correlationID := event.ID()
	extensions := event.Extensions()
	if extensions != nil {
		if cid, ok := extensions[mq.CECorrelIDAttr]; ok {
			correlationID = cid.(string)
		}
	}

	if err := a.queue.Put(msg, correlationID); err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	a.sr.ReportProcessingSuccess(ceTypeTag, ceSrcTag)
	return a.replier.Ok(&event, "ok")
}
