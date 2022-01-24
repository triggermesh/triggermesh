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

package jqtransformation

import (
	"context"

	"github.com/itchyny/gojq"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig
	// Query represents the jq query to be applied to the incoming event
	Query string `envconfig:"QUERY" required:"true"`
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
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType("io.triggermesh.jqtransformation.error"),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	query, err := gojq.Parse(env.Query)
	if err != nil {
		logger.Panicf("Error creating query: %v", err)
	}

	return &Adapter{
		query: query,

		sink:     env.Sink,
		replier:  replier,
		ceClient: ceClient,
		logger:   logger,
	}
}

var _ pkgadapter.Adapter = (*Adapter)(nil)

type Adapter struct {
	query *gojq.Query

	sink     string
	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Start is a blocking function and will return if an error occurs
// or the context is cancelled.
func (a *Adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting JQTransformation Adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *Adapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var data map[string]interface{}
	var qd interface{}
	if err := event.DataAs(&data); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	iter := a.query.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}
		qd = v
	}

	if err := event.SetData(cloudevents.ApplicationJSON, qd); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	if a.sink != "" {
		if err := a.ceClient.Send(ctx, event); err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
		return nil, cloudevents.ResultACK
	}

	return &event, cloudevents.ResultACK
}
