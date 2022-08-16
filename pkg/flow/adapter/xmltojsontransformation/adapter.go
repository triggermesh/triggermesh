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

package xmltojsontransformation

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	xj "github.com/basgys/goxml2json"

	"github.com/triggermesh/triggermesh/pkg/apis/flow"
	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
	// Sink defines the target sink for the events. If no Sink is defined the
	// events are replied back to the sender.
	Sink string `envconfig:"K_SINK"`
}

// NewAdapter adapter implementation
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: flow.XMLToJSONTransformationResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeXMLToJSONGenericResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &Adapter{
		sink:     env.Sink,
		replier:  replier,
		ceClient: ceClient,
		logger:   logger,

		mt: mt,
		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*Adapter)(nil)

type Adapter struct {
	sink     string
	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	mt *pkgadapter.MetricTag
	sr *metrics.EventProcessingStatsReporter
}

// Start is a blocking function and will return if an error occurs
// or the context is cancelled.
func (a *Adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting XMLToJSONTransformation Adapter")
	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *Adapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	if !isValidXML(event.Data()) {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation,
			errors.New("invalid XML"), nil)
	}

	xml := bytes.NewReader(event.Data())
	jsn, err := xj.Convert(xml)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	readBuf, err := io.ReadAll(jsn)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	if err := event.SetData(cloudevents.ApplicationJSON, readBuf); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	if a.sink != "" {
		if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
		return nil, cloudevents.ResultACK
	}

	return &event, cloudevents.ResultACK
}

func isValidXML(data []byte) bool {
	return xml.Unmarshal(data, new(interface{})) == nil
}
