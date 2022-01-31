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

package azureeventhubstarget

import (
	"context"
	"encoding/json"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeAzureEventHubsGenericResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	hub, err := eventhub.NewHubFromEnvironment()
	if err != nil {
		logger.Panicf("Error creating EventHub connection: %v", err)
	}

	return &adapter{
		hub: hub,

		discardCEContext: env.DiscardCEContext,
		replier:          replier,
		ceClient:         ceClient,
		logger:           logger,
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

type adapter struct {
	hub *eventhub.Hub

	discardCEContext bool
	replier          *targetce.Replier
	ceClient         cloudevents.Client
	logger           *zap.SugaredLogger
}

// Returns if stopCh is closed or Send() returns an error.
func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Azure EventHub Target adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *adapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	if a.discardCEContext {
		err := a.hub.Send(ctx, eventhub.NewEvent(event.Data()))
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
	} else {
		// Serialize the event first, and then stream it
		bs, err := json.Marshal(event)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
		err = a.hub.Send(ctx, eventhub.NewEvent(bs))
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
	}

	return a.replier.Ok(&event, "ok")
}
