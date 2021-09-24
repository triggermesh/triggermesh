/*
Copyright 2021 TriggerMesh Inc.

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

package googlecloudpubsubsource

import (
	"context"
	"fmt"
	"strconv"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	CESource string `envconfig:"CE_SOURCE" required:"true"`

	ProjectID         string `envconfig:"GCLOUD_PROJECT" required:"true"`
	ServiceAccountKey []byte `envconfig:"GCLOUD_SERVICEACCOUNT_KEY" required:"true"`

	SubscriptionID string `envconfig:"GCLOUD_PUBSUB_SUBSCRIPTION" required:"true"`

	// Name of a message processor which takes care of converting Pub/Sub
	// messages to CloudEvents.
	//
	// Supported values: [ default ]
	MessageProcessor string `envconfig:"GCLOUD_PUBSUB_MESSAGE_PROCESSOR" default:"default"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger   *zap.SugaredLogger
	ceClient cloudevents.Client
	subs     *pubsub.Subscription
	msgPrcsr MessageProcessor
}

var _ pkgadapter.Adapter = (*adapter)(nil)

// NewEnvConfig returns an accessor for the source's adapter envConfig.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter returns a constructor for the source's adapter.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	env := envAcc.(*envConfig)

	psCli, err := pubsub.NewClient(ctx, env.ProjectID, option.WithCredentialsJSON(env.ServiceAccountKey))
	if err != nil {
		logger.Panicw("Failed to create Google Cloud Pub/Sub API client", zap.Error(err))
	}

	var msgPrcsr MessageProcessor
	switch env.MessageProcessor {
	case "default":
		msgPrcsr = &defaultMessageProcessor{ceSource: env.CESource}
	default:
		panic("unsupported message processor " + strconv.Quote(env.MessageProcessor))
	}

	return &adapter{
		logger:   logger,
		ceClient: ceClient,
		subs:     psCli.Subscription(env.SubscriptionID),
		msgPrcsr: msgPrcsr,
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	err := a.subs.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		events, err := a.msgPrcsr.Process(msg)
		if err != nil {
			a.logger.Errorw("Failed to process Pub/Sub message", zap.Error(err))
			msg.Nack()
		}

		var sendErrs errList

		for _, event := range events {
			if result := a.ceClient.Send(ctx, *event); !cloudevents.IsACK(result) {
				sendErrs.errs = append(sendErrs.errs, err)
				continue
			}
		}

		if len(sendErrs.errs) != 0 {
			a.logger.Errorw("Failed to send CloudEvents", zap.Error(sendErrs))
			msg.Nack()
		}

		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("during runtime of message receiver: %w", err)
	}

	return nil
}
