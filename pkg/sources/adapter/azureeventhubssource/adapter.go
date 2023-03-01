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

package azureeventhubssource

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/devigned/tab"
	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/protocol"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/azureeventhubssource/trace"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/health"
)

const (
	connTimeout  = 20 * time.Second
	drainTimeout = 1 * time.Minute
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	// Resource ID of the Event Hubs instance.
	// Used to set the 'source' context attribute of CloudEvents.
	HubResourceID string `envconfig:"EVENTHUB_RESOURCE_ID" required:"true"`

	// Consumer group name to be used by the source.
	ConsumerGroup string `envconfig:"EVENTHUB_CONSUMER_GROUP"`

	// Name of a message processor which takes care of converting Event
	// Hubs messages to CloudEvents.
	//
	// Supported values: [ default eventgrid ]
	MessageProcessor string `envconfig:"EVENTHUB_MESSAGE_PROCESSOR" default:"default"`

	// Allows overriding common CloudEvents attributes.
	CEOverrideSource string `envconfig:"CE_SOURCE"`
	CEOverrideType   string `envconfig:"CE_TYPE"`

	// The environment variables below aren't read from the envConfig struct
	// by the Event Hubs SDK, but rather directly using os.Getenv().
	// They are nevertheless listed here for documentation purposes.
	_ string `envconfig:"EVENTHUB_NAMESPACE"`
	_ string `envconfig:"EVENTHUB_NAME"`
	_ string `envconfig:"AZURE_TENANT_ID"`
	_ string `envconfig:"AZURE_CLIENT_ID"`
	_ string `envconfig:"AZURE_CLIENT_SECRET"`
	_ string `envconfig:"EVENTHUB_KEY_NAME"`
	_ string `envconfig:"EVENTHUB_KEY_VALUE"`
	_ string `envconfig:"EVENTHUB_CONNECTION_STRING"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger *zap.SugaredLogger
	mt     *pkgadapter.MetricTag

	runtimeInfo *eventhub.HubRuntimeInformation

	ehClient *eventhub.Hub
	ceClient cloudevents.Client

	ehConsumerGroup string
	msgPrcsr        MessageProcessor
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		// TODO(antoineco): This adapter is used by multiple kinds. Set ResourceGroup based on actual kind.
		ResourceGroup: sources.AzureEventHubsSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	hub, err := eventhub.NewHubFromEnvironment()
	if err != nil {
		logger.Panicw("Unable to create Event Hub client", zap.Error(err))
	}

	ceSource := env.HubResourceID
	if ceOverrideSource := env.CEOverrideSource; ceOverrideSource != "" {
		ceSource = ceOverrideSource
	}

	ceType := v1alpha1.AzureEventType(sources.AzureServiceEventHub, v1alpha1.AzureEventHubGenericEventType)
	if ceOverrideType := env.CEOverrideType; ceOverrideType != "" {
		ceType = ceOverrideType
	}

	var msgPrcsr MessageProcessor
	switch env.MessageProcessor {
	case "eventgrid":
		msgPrcsr = &eventGridMessageProcessor{
			ceSourceFallback: ceSource,
			ceTypeFallback:   ceType,
		}
	case "default":
		msgPrcsr = &defaultMessageProcessor{
			ceSource: ceSource,
			ceType:   ceType,
		}
	default:
		panic("Unsupported message processor " + strconv.Quote(env.MessageProcessor))
	}

	consumerGroup := eventhub.DefaultConsumerGroup
	if env.ConsumerGroup != "" {
		consumerGroup = env.ConsumerGroup
	}

	// The Event Hubs client uses the default "NoOpTracer" tab.Tracer
	// implementation, which does not produce any log message. We register
	// a custom implementation so that event handling errors are logged via
	// Knative's logging facilities.
	tab.Register(trace.NewNoOpTracerWithLogger(logger))

	return &adapter{
		logger: logger,
		mt:     mt,

		ceClient: ceClient,
		ehClient: hub,

		ehConsumerGroup: consumerGroup,

		msgPrcsr: msgPrcsr,
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	connCtx, cancel := context.WithTimeout(ctx, connTimeout)
	runtimeInfo, err := a.ehClient.GetRuntimeInformation(connCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("getting Event Hub runtime information: %w", err)
	}
	a.runtimeInfo = runtimeInfo

	a.logger.Info("Starting Event Hub message receivers for partitions ", runtimeInfo.PartitionIDs)

	// TODO(antoineco): Find a way to inject Prometheus metric tags into
	// the context.Context that is passed to handleMessage().
	// Currently, the SDK always passes context.Background(), instead of our ctx:
	// https://github.com/Azure/azure-event-hubs-go/blob/v3.3.17/receiver.go#L219
	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	// listen to each partition of the Event Hub
	for _, partitionID := range runtimeInfo.PartitionIDs {
		connCtx, cancel := context.WithTimeout(ctx, connTimeout)
		_, err := a.ehClient.Receive(connCtx, partitionID, a.handleMessage,
			eventhub.ReceiveWithLatestOffset(), eventhub.ReceiveWithConsumerGroup(a.ehConsumerGroup))
		cancel()
		if err != nil {
			a.logger.Errorw("An error occurred while starting message receivers. "+
				"Terminating all active receivers", zap.Error(err))

			closeCtx, cancel := context.WithTimeout(ctx, drainTimeout)
			defer cancel()
			if err := a.ehClient.Close(closeCtx); err != nil {
				a.logger.Errorw("An additional error occurred while terminating active "+
					"Event Hub message receivers", zap.Error(err))
			}

			return fmt.Errorf("starting message receiver for partition %s: %w", partitionID, err)
		}
	}

	health.MarkReady()

	<-ctx.Done()
	a.logger.Debug("Terminating all active Event Hub message receivers")

	closeCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()
	if err := a.ehClient.Close(closeCtx); err != nil {
		return fmt.Errorf("terminating active Event Hub message receivers: %w", err)
	}

	return nil
}

// handleMessage satisfies eventhub.Handler.
func (a *adapter) handleMessage(ctx context.Context, msg *eventhub.Event) error {
	if msg == nil {
		return nil
	}

	events, err := a.msgPrcsr.Process(msg)
	if err != nil {
		return fmt.Errorf("processing Event Hubs message with ID %s: %w", msg.ID, err)
	}

	var sendErrs errList

	for _, ev := range events {
		if err := ev.Validate(); err != nil {
			ev = sanitizeEvent(err.(event.ValidationError), ev)
		}

		if err := sendCloudEvent(ctx, a.ceClient, ev); err != nil {
			sendErrs.errs = append(sendErrs.errs,
				fmt.Errorf("failed to send event with ID %s: %w", ev.ID(), err),
			)
			continue
		}
	}

	if len(sendErrs.errs) != 0 {
		return fmt.Errorf("sending events to the sink: %w", sendErrs)
	}

	return nil
}

// sendCloudEvent sends a single CloudEvent to the event sink.
func sendCloudEvent(ctx context.Context, cli cloudevents.Client, event *cloudevents.Event) protocol.Result {
	if result := cli.Send(ctx, *event); !cloudevents.IsACK(result) {
		return result
	}
	return nil
}

// errList is an aggregate of errors.
type errList struct {
	errs []error
}

var _ error = (*errList)(nil)

// Error implements the error interface.
func (e errList) Error() string {
	if len(e.errs) == 0 {
		return ""
	}
	return fmt.Sprintf("%q", e.errs)
}

// sanitizeEvent tries to fix the validation issues listed in the given
// cloudevents.ValidationError, and returns a sanitized version of the event.
//
// For now, this helper exists solely to fix CloudEvents sent by Azure Event
// Grid, which often contain
//
//	"dataschema": "#"
func sanitizeEvent(validErrs event.ValidationError, origEvent *cloudevents.Event) *cloudevents.Event {
	for attr := range validErrs {
		// we don't bother cloning, events are garbage collected after
		// being sent to the sink
		switch attr {
		case "dataschema":
			origEvent.SetDataSchema("")
		}
	}

	return origEvent
}
