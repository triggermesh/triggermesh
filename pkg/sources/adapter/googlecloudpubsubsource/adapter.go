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

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	SubscriptionResourceName GCloudResourceName `envconfig:"GCLOUD_PUBSUB_SUBSCRIPTION" required:"true"`

	ServiceAccountKey []byte `envconfig:"GCLOUD_SERVICEACCOUNT_KEY" required:"false"`

	// Allows overriding common CloudEvents attributes.
	CEOverrideSource string `envconfig:"CE_SOURCE"`
	CEOverrideType   string `envconfig:"CE_TYPE"`

	// Name of a message processor which takes care of converting Pub/Sub
	// messages to CloudEvents.
	//
	// Supported values: [ default ]
	MessageProcessor string `envconfig:"GCLOUD_PUBSUB_MESSAGE_PROCESSOR" default:"default"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger   *zap.SugaredLogger
	mt       *pkgadapter.MetricTag
	ceClient cloudevents.Client
	subs     *pubsub.Subscription
	msgPrcsr MessageProcessor
}

var _ pkgadapter.Adapter = (*adapter)(nil)

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		// TODO(antoineco): This adapter is used by multiple kinds. Set ResourceGroup based on actual kind.
		ResourceGroup: sources.GoogleCloudPubSubSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	opts := make([]option.ClientOption, 0)
	if env.ServiceAccountKey != nil {
		opts = append(opts, option.WithCredentialsJSON(env.ServiceAccountKey))
	}
	psCli, err := pubsub.NewClient(ctx, env.SubscriptionResourceName.Project, opts...)
	if err != nil {
		logger.Panicw("Failed to create Google Cloud Pub/Sub API client", zap.Error(err))
	}

	subsCli := psCli.Subscription(env.SubscriptionResourceName.Resource)

	sub, err := subsCli.Config(ctx)
	if err != nil {
		logger.Panicw("Failed to read configuration of Pub/Sub Subscription "+
			strconv.Quote(env.SubscriptionResourceName.String()), zap.Error(err))
	}
	topicName := sub.Topic.String()

	ceSource := topicName
	if ceOverrideSource := env.CEOverrideSource; ceOverrideSource != "" {
		ceSource = ceOverrideSource
	}

	ceType := v1alpha1.GoogleCloudPubSubGenericEventType
	if ceOverrideType := env.CEOverrideType; ceOverrideType != "" {
		ceType = ceOverrideType
	}

	var msgPrcsr MessageProcessor
	switch env.MessageProcessor {
	case "default":
		msgPrcsr = &defaultMessageProcessor{
			ceSource: ceSource,
			ceType:   ceType,
		}
	case "gcs":
		msgPrcsr = &gcsMessageProcessor{
			ceSource: ceSource,
		}
	default:
		logger.Panic("Unsupported message processor " + strconv.Quote(env.MessageProcessor))
	}

	return &adapter{
		logger:   logger,
		mt:       mt,
		ceClient: ceClient,
		subs:     subsCli,
		msgPrcsr: msgPrcsr,
	}
}

// Start implements adapter.Adapter.
// Required permissions:
// - pubsub.subscriptions.consume
func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting message receiver")

	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	if err := a.subs.Receive(ctx, a.handleMessage); err != nil {
		return fmt.Errorf("during runtime of message receiver: %w", err)
	}

	return nil
}

// handleMessage is called by the receiver whenever a Message is pulled from
// the Pub/Sub subscription.
func (a *adapter) handleMessage(ctx context.Context, msg *pubsub.Message) {
	events, err := a.msgPrcsr.Process(msg)
	if err != nil {
		a.logger.Errorw("Failed to process Pub/Sub message", zap.Error(err))
		msg.Nack()
	}

	var sendErrs errList

	for _, event := range events {
		if result := a.ceClient.Send(ctx, *event); !cloudevents.IsACK(result) {
			sendErrs.errs = append(sendErrs.errs,
				fmt.Errorf("failed to send event with ID %s: %w", event.ID(), result),
			)
			continue
		}
	}

	if len(sendErrs.errs) != 0 {
		a.logger.Errorw("Failed to send CloudEvents", zap.Error(sendErrs))
		msg.Nack()
	}

	msg.Ack()
}
