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

package synchronizer

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

var _ pkgadapter.Adapter = (*adapter)(nil)

type adapter struct {
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	correlationKey  *correlationKey
	responseTimeout time.Duration

	sessions *storage
	sinkURL  string
	bridgeID string
}

// NewAdapter returns adapter implementation.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	key, err := newCorrelationKey(env.CorrelationKey, env.CorrelationKeyLength)
	if err != nil {
		logger.Panic("Cannot create an instance of Correlation Key: %v", err)
	}

	return &adapter{
		ceClient:       ceClient,
		logger:         logger,
		correlationKey: key,

		responseTimeout: env.ResponseWaitTimeout,
		sessions:        newStorage(),
		sinkURL:         env.Sink,
		bridgeID:        env.BridgeIdentifier,
	}
}

// Returns if stopCh is closed or Send() returns an error.
func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Synchronizer Adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *adapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	a.logger.Debugf("Received the event: %s", event.String())

	if correlationID, exists := a.correlationKey.get(event); exists {
		return a.serveResponse(ctx, correlationID, event)
	}

	correlationID := a.correlationKey.set(&event)
	return a.serveRequest(ctx, correlationID, event)
}

// serveRequest creates the session for the incoming events and blocks the client.
func (a *adapter) serveRequest(ctx context.Context, correlationID string, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	a.logger.Debugf("Handling request %q", correlationID)

	respChan, err := a.sessions.add(correlationID)
	if err != nil {
		return nil, cloudevents.NewHTTPResult(http.StatusInternalServerError, "cannot add session %q: %w", correlationID, err)
	}
	defer a.sessions.delete(correlationID)

	sendErr := make(chan error)
	defer close(sendErr)

	go func(errChan chan error) {
		if res := a.ceClient.Send(cloudevents.ContextWithTarget(ctx, a.sinkURL), a.withBridgeIdentifier(&event)); cloudevents.IsUndelivered(res) {
			errChan <- res
		}
	}(sendErr)

	a.logger.Debugf("Request forwarded to %q", a.sinkURL)

	t := time.NewTimer(a.responseTimeout)
	defer t.Stop()

	a.logger.Debugf("Waiting response for %q", correlationID)

	select {
	case err := <-sendErr:
		a.logger.Errorf("Unable to forward the request: %v", err)
		return nil, cloudevents.NewHTTPResult(http.StatusBadRequest, "unable to forward the request: %v", err)
	case result := <-respChan:
		if result == nil {
			a.logger.Errorf("No response from %q", correlationID)
			return nil, cloudevents.NewHTTPResult(http.StatusInternalServerError, "failed to communicate the response")
		}
		a.logger.Debugf("Received response for %q", correlationID)
		res := a.withBridgeIdentifier(result)
		return &res, cloudevents.ResultACK
	case <-t.C:
		a.logger.Errorf("Request %q timed out", correlationID)
		return nil, cloudevents.NewHTTPResult(http.StatusGatewayTimeout, "backend did not respond in time")
	}
}

// serveResponse matches event's correlation key and writes response back to the session's communication channel.
func (a *adapter) serveResponse(ctx context.Context, correlationID string, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	a.logger.Debugf("Handling response %q", correlationID)

	responseChan, exists := a.sessions.get(correlationID)
	if !exists {
		a.logger.Errorf("Session for %q does not exist", correlationID)
		return nil, cloudevents.NewHTTPResult(http.StatusBadGateway, "client session does not exist")
	}

	a.logger.Debugf("Forwarding response %q", correlationID)

	select {
	case responseChan <- &event:
		a.logger.Debugf("Response %q completed", correlationID)
		return nil, cloudevents.ResultACK
	default:
		a.logger.Errorf("Unable to forward the response %q", correlationID)
		return nil, cloudevents.NewHTTPResult(http.StatusBadGateway, "client connection is closed")
	}
}

// withBridgeIdentifier adds Bridge ID to the event context.
func (a *adapter) withBridgeIdentifier(event *cloudevents.Event) cloudevents.Event {
	if a.bridgeID == "" {
		return *event
	}
	if bid, err := event.Context.GetExtension(targetce.StatefulWorkflowHeader); err != nil && bid != "" {
		return *event
	}
	event.SetExtension(targetce.StatefulWorkflowHeader, a.bridgeID)
	return *event
}
