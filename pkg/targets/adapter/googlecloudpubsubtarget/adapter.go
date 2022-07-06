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

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	TopicName GCloudResourceName `envconfig:"GCLOUD_PUBSUB_TOPIC" required:"true"`

	ServiceAccountKey []byte `envconfig:"GCLOUD_SERVICEACCOUNT_KEY" required:"true"`

	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
}

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType("io.triggermesh.googlecloudpubsubtarget.response"),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	psCli, err := pubsub.NewClient(ctx, env.TopicName.Project,
		option.WithCredentialsJSON(env.ServiceAccountKey),
	)
	if err != nil {
		logger.Panicw("Failed to create Google Cloud Pub/Sub API client", zap.Error(err))
	}

	t := psCli.Topic(env.TopicName.Resource)
	return &googlecloudpubsubtargetAdapter{
		topic:    t,
		replier:  replier,
		ceClient: ceClient,
		logger:   logger,
	}
}

var _ pkgadapter.Adapter = (*googlecloudpubsubtargetAdapter)(nil)

type googlecloudpubsubtargetAdapter struct {
	topic *pubsub.Topic

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Returns if stopCh is closed or Send() returns an error.
func (a *googlecloudpubsubtargetAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting GoogleCloudPubSubTarget Adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *googlecloudpubsubtargetAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	result := a.topic.Publish(ctx, &pubsub.Message{
		Data: event.Data(),
	})
	id, err := result.Get(ctx)
	if err != nil {
		a.logger.Panic(err)
	}
	a.logger.Infof("Published a message; msg ID: %v\n", id)

	return a.replier.Ok(&event, "ok")
}
