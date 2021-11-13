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

package azureeventhubsource

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

	"github.com/triggermesh/triggermesh/pkg/sources/adapter/azureeventhubsource/trace"
)

// Event Hub connection connTimeout
const connTimeout = 20 * time.Second

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	// Resource ID of the Event Hubs instance.
	// Used to set the 'source' context attribute of CloudEvents.
	HubResourceID string `envconfig:"EVENTHUB_RESOURCE_ID" required:"true"`

	// Name of a message processor which takes care of converting Event
	// Hubs messages to CloudEvents.
	//
	// Supported values: [ default eventgrid ]
	MessageProcessor string `envconfig:"EVENTHUB_MESSAGE_PROCESSOR" default:"default"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger *zap.SugaredLogger

	runtimeInfo *eventhub.HubRuntimeInformation

	ehClient *eventhub.Hub
	ceClient cloudevents.Client

	msgPrcsr MessageProcessor
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	env := envAcc.(*envConfig)

	hub, err := eventhub.NewHubFromEnvironment()
	if err != nil {
		logger.Panicw("Unable to create Event Hub client", zap.Error(err))
	}

	ceSource := env.HubResourceID

	var msgPrcsr MessageProcessor
	switch env.MessageProcessor {
	case "eventgrid":
		msgPrcsr = &eventGridMessageProcessor{ceSourceFallback: ceSource}
	case "default":
		msgPrcsr = &defaultMessageProcessor{ceSource: ceSource}
	default:
		panic("Unsupported message processor " + strconv.Quote(env.MessageProcessor))
	}

	// The Event Hubs client uses the default "NoOpTracer" tab.Tracer
	// implementation, which does not produce any log message. We register
	// a custom implementation so that event handling errors are logged via
	// Knative's logging facilities.
	tab.Register(trace.NewNoOpTracerWithLogger(logger))

	return &adapter{
		logger: logger,

		ehClient: hub,
		ceClient: ceClient,

		msgPrcsr: msgPrcsr,
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	connCtx, cancel := context.WithTimeout(ctx, connTimeout)
	runtimeInfo, err := a.ehClient.GetRuntimeInformation(connCtx)
	cancel()
	if err != nil {
		a.logger.Fatal("GetRuntimeInformation failed:", err)
	}
	a.runtimeInfo = runtimeInfo

	// listen to each partition of the Event Hub
	for _, partitionID := range runtimeInfo.PartitionIDs {
		connCtx, cancel := context.WithTimeout(ctx, connTimeout)
		_, err := a.ehClient.Receive(connCtx, partitionID, a.handleMessage, eventhub.ReceiveWithLatestOffset())
		cancel()
		if err != nil {
			a.logger.Error("failed to start Event Hub listener:", err)
			continue
		}
	}

	<-ctx.Done()

	return a.ehClient.Close(context.Background())
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
//   "dataschema": "#"
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
