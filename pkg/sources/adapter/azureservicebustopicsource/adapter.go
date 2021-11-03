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

package azureservicebustopicsource

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/devigned/tab"
	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/protocol"

	servicebus "github.com/Azure/azure-service-bus-go"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/azureservicebustopicsource/trace"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	// Resource ID of the Service Bus Subscription.
	SubsResourceID string `envconfig:"SERVICEBUS_SUBSCRIPTION_RESOURCE_ID" required:"true"`

	// Name of a message processor which takes care of converting Service
	// Bus messages to CloudEvents.
	//
	// Supported values: [ default ]
	MessageProcessor string `envconfig:"SERVICEBUS_MESSAGE_PROCESSOR" default:"default"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger *zap.SugaredLogger

	msgRcvr  *servicebus.Receiver
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

	subsResourceID, err := parseSubscriptionResID(env.SubsResourceID)
	if err != nil {
		logger.Panicw("Unable to parse Subscription ID", zap.Error(err))
	}

	nsName := subsResourceID.Namespace
	topicName := subsResourceID.ResourceName
	subsName := subsResourceID.SubResourceName

	ns, err := servicebus.NewNamespace(servicebus.NamespaceWithEnvironmentBinding(nsName))
	if err != nil {
		logger.Panicw("Unable to obtain interface for Service Bus Namespace", zap.Error(err))
	}

	subsEntityPath := topicName + "/Subscriptions/" + subsName
	rcvr, err := ns.NewReceiver(ctx, subsEntityPath)
	if err != nil {
		logger.Panicw("Unable to obtain message receiver for Service Bus Subscription", zap.Error(err))
	}

	ceSource := env.SubsResourceID

	var msgPrcsr MessageProcessor
	switch env.MessageProcessor {
	case "default":
		msgPrcsr = &defaultMessageProcessor{ceSource: ceSource}
	default:
		panic("unsupported message processor " + strconv.Quote(env.MessageProcessor))
	}

	// The Service Bus client uses the default "NoOpTracer" tab.Tracer
	// implementation, which does not produce any log message. We register
	// a custom implementation so that event handling errors are logged via
	// Knative's logging facilities.
	tab.Register(trace.NewNoOpTracerWithLogger(logger))

	return &adapter{
		logger: logger,

		ceClient: ceClient,

		msgRcvr:  rcvr,
		msgPrcsr: msgPrcsr,
	}
}

// parseSubscriptionResID parses the given Subscription resource ID string to a
// structured resource ID.
func parseSubscriptionResID(resIDStr string) (*v1alpha1.AzureResourceID, error) {
	resID := &v1alpha1.AzureResourceID{}

	err := json.Unmarshal([]byte(strconv.Quote(resIDStr)), resID)
	if err != nil {
		return nil, fmt.Errorf("deserializing resource ID string: %w", err)
	}

	return resID, nil
}

// Start implements adapter.Adapter.
// Required permissions:
//  - Microsoft.ServiceBus/namespaces/topics/read
//  - Microsoft.ServiceBus/namespaces/topics/subscriptions/read
func (a *adapter) Start(ctx context.Context) error {
	handle := a.msgRcvr.Listen(ctx, servicebus.HandlerFunc(a.handleMessage))
	<-handle.Done()
	return handle.Err()
}

// handleMessage satisfies servicebus.HandlerFunc.
func (a *adapter) handleMessage(ctx context.Context, msg *servicebus.Message) error {
	if msg == nil {
		return nil
	}

	events, err := a.msgPrcsr.Process(msg)
	if err != nil {
		return fmt.Errorf("processing Service Bus message with ID %s: %w", msg.ID, err)
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
