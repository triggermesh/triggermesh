/*
Copyright 2020 TriggerMesh Inc.

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

package salesforcesource

import (
	"context"
	"encoding/json"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/sources/adapter/salesforcesource/auth"
	sfclient "github.com/triggermesh/triggermesh/pkg/sources/adapter/salesforcesource/client"
)

const eventType = "com.salesforce.stream.message"

type salesforceAdapter struct {
	sfVersion         string
	sfChannel         string
	sfInitialReplayID int

	sfAuth auth.Authenticator

	dispatcher *eventDispatcher
	logger     *zap.SugaredLogger
}

type eventDispatcher struct {
	eventSource string

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

var _ adapter.Adapter = (*salesforceAdapter)(nil)
var _ sfclient.EventDispatcher = (*eventDispatcher)(nil)

// NewAdapter implementation
func NewAdapter(ctx context.Context, aEnv adapter.EnvConfigAccessor, ceClient cloudevents.Client) adapter.Adapter {
	env := aEnv.(*envAccessor)
	logger := logging.FromContext(ctx)

	source := env.Name
	if env.SubscriptionChannel[0] != '/' {
		source += "/"
	}
	source += env.SubscriptionChannel

	dispatcher := &eventDispatcher{
		eventSource: source,
		ceClient:    ceClient,
		logger:      logger.Named("dispatcher"),
	}

	jwtAuth, err := auth.NewJWTAuthenticator(env.CertKey, env.ClientID, env.User, env.AuthServer, http.DefaultClient, logger.Named("authenticator"))
	if err != nil {
		logger.Panic(err)
	}

	adapter := &salesforceAdapter{
		sfVersion:         env.Version,
		sfChannel:         env.SubscriptionChannel,
		sfInitialReplayID: env.SubscriptionReplayID,
		sfAuth:            jwtAuth,

		dispatcher: dispatcher,
		logger:     logger,
	}

	return adapter
}

// Start runs the handler.
func (a *salesforceAdapter) Start(ctx context.Context) (err error) {
	replayID := a.sfInitialReplayID

	subs := []sfclient.Subscription{
		{
			Channel:  a.sfChannel,
			ReplayID: replayID,
		},
	}

	client := sfclient.NewBayeux(a.sfVersion, subs, a.sfAuth, a.dispatcher, http.DefaultClient, a.logger.Named("bayeux"))

	return client.Start(ctx)
}

func (e *eventDispatcher) DispatchEvent(ctx context.Context, msg *sfclient.ConnectResponse) {
	event := cloudevents.NewEvent(cloudevents.VersionV1)

	event.SetType(eventType)
	event.SetSource(e.eventSource)
	event.SetID(uuid.New().String())
	event.SetSubject(subjectNameFromConnectResponse(msg))
	if err := event.SetData(cloudevents.ApplicationJSON, msg.Data); err != nil {
		e.logger.Error("Failed to set event data: %w", err)
		return
	}

	if result := e.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		e.logger.Errorf("Could not send CloudEvent %s: %v", event.Subject(), result)
		return
	}
}

func (e *eventDispatcher) DispatchError(err error) {
	e.logger.Errorf("Error receiving events: %v", err)
}

func subjectNameFromConnectResponse(msg *sfclient.ConnectResponse) string {

	// if ChangeDataCapture look for entity/operation
	cdc := &sfclient.ChangeDataCapturePayload{}
	if err := json.Unmarshal(msg.Data.Payload, cdc); err == nil {
		return cdc.ChangeEventHeader.EntityName + "/" + cdc.ChangeEventHeader.ChangeType
	}

	// if PushTopic look for object-name/event-operation
	ptso := &sfclient.PushTopicSObject{}
	if err := json.Unmarshal(msg.Data.Payload, ptso); err == nil {
		return ptso.Name + "/" + msg.Data.Event.Type
	}

	// default to channel
	return msg.Channel
}
