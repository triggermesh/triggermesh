/*
Copyright (c) 2021 TriggerMesh Inc.

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

package logztarget

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/logzio/logzio-go"
	cloudevents2 "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
	"go.uber.org/zap"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	replier, err := cloudevents2.New(env.Component, logger.Named("replier"),
		cloudevents2.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		cloudevents2.ReplierWithStaticResponseType(v1alpha1.EventTypeLogzShipResponse),
		cloudevents2.ReplierWithPayloadPolicy(cloudevents2.PayloadPolicy(env.CloudEventPayloadPolicy)))

	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	l, err := logzio.New(
		env.ShippingToken,
		logzio.SetUrl("https://"+env.LogsListenerURL+":8071"),
	)
	if err != nil {
		panic(err)
	}

	return &logzAdapter{
		l: l,

		replier:  replier,
		ceClient: ceClient,
		logger:   logger,
	}
}

var _ pkgadapter.Adapter = (*logzAdapter)(nil)

type logzAdapter struct {
	l *logzio.LogzioSender

	replier  *cloudevents2.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Returns if stopCh is closed or Send() returns an error.
func (a *logzAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Logz adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *logzAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	err := a.l.Send(event.Data())
	if err != nil {
		return a.replier.Error(&event, cloudevents2.ErrorCodeAdapterProcess, err, nil)
	}

	return a.replier.Ok(&event, "ok")
}
