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

package azureservicebussource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/devigned/tab"
	"go.uber.org/zap"
	"nhooyr.io/websocket"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/protocol"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/azureservicebussource/trace"
)

const (
	resourceProviderServiceBus = "Microsoft.ServiceBus"

	resourceTypeQueues        = "queues"
	resourceTypeTopics        = "topics"
	resourceTypeSubscriptions = "subscriptions"
)

const (
	envKeyName  = "SERVICEBUS_KEY_NAME"
	envKeyValue = "SERVICEBUS_KEY_VALUE"
	envConnStr  = "SERVICEBUS_CONNECTION_STRING"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	// Resource ID of the Service Bus entity (Queue or Topic subscription).
	EntityResourceID string `envconfig:"SERVICEBUS_ENTITY_RESOURCE_ID" required:"true"`

	// Name of a message processor which takes care of converting Service
	// Bus messages to CloudEvents.
	//
	// Supported values: [ default ]
	MessageProcessor string `envconfig:"SERVICEBUS_MESSAGE_PROCESSOR" default:"default"`

	// WebSocketsEnable.
	WebSocketsEnable bool `envconfig:"SERVICEBUS_WEBSOCKETS_ENABLE" required:"false"`

	// The environment variables below aren't read from the envConfig struct
	// by the Service Bus SDK, but rather directly using os.Getenv().
	// They are nevertheless listed here for documentation purposes.
	_ string `envconfig:"AZURE_TENANT_ID"`
	_ string `envconfig:"AZURE_CLIENT_ID"`
	_ string `envconfig:"AZURE_CLIENT_SECRET"`
	_ string `envconfig:"SERVICEBUS_KEY_NAME"`
	_ string `envconfig:"SERVICEBUS_KEY_VALUE"`
	_ string `envconfig:"SERVICEBUS_CONNECTION_STRING"`
}

// adapter implements the source's adapter.
type adapter struct {
	mt *pkgadapter.MetricTag

	msgRcvr  *azservicebus.Receiver
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

	mt := &pkgadapter.MetricTag{
		Namespace: envAcc.GetNamespace(),
		Name:      envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	entityID, err := parseServiceBusResourceID(env.EntityResourceID)
	if err != nil {
		logger.Panicw("Unable to parse entity ID "+strconv.Quote(env.EntityResourceID), zap.Error(err))
	}

	client, err := clientFromEnvironment(entityID, webSocketsClientOptions(env.WebSocketsEnable))
	if err != nil {
		logger.Panicw("Unable to obtain interface for Service Bus Namespace", zap.Error(err))
	}

	var rcvr *azservicebus.Receiver
	switch entityID.ResourceType {
	case resourceTypeQueues:
		rcvr, err = client.NewReceiverForQueue(entityID.ResourceName, nil)
		mt.ResourceGroup = sources.AzureServiceBusQueueSourceResource.String()
	case resourceTypeSubscriptions, resourceTypeTopics:
		rcvr, err = client.NewReceiverForSubscription(entityID.ResourceName, entityID.SubResourceName, nil)
		mt.ResourceGroup = sources.AzureServiceBusTopicSourceResource.String()
	}
	if err != nil {
		logger.Panicw("Unable to obtain message receiver for Service Bus entity "+strconv.Quote(strconv.Quote(entityPath(entityID))), zap.Error(err))
	}

	ceSource := env.EntityResourceID

	var msgPrcsr MessageProcessor
	switch env.MessageProcessor {
	case "default":
		msgPrcsr = &defaultMessageProcessor{ceSource: ceSource}
	default:
		logger.Panic("unsupported message processor " + strconv.Quote(env.MessageProcessor))
	}

	// The Service Bus client uses the default "NoOpTracer" tab.Tracer
	// implementation, which does not produce any log message. We register
	// a custom implementation so that event handling errors are logged via
	// Knative's logging facilities.
	tab.Register(trace.NewNoOpTracerWithLogger(logger))

	return &adapter{
		mt: mt,

		ceClient: ceClient,

		msgRcvr:  rcvr,
		msgPrcsr: msgPrcsr,
	}
}

// parseServiceBusResourceID parses the given resource ID string to a
// structured resource ID, and validates that this resource ID refers to a
// Service Bus entity.
func parseServiceBusResourceID(resIDStr string) (*v1alpha1.AzureResourceID, error) {
	resID := &v1alpha1.AzureResourceID{}

	err := json.Unmarshal([]byte(strconv.Quote(resIDStr)), resID)
	if err != nil {
		return nil, fmt.Errorf("deserializing resource ID string: %w", err)
	}

	// Must match one of the following patterns:
	//  - /.../providers/Microsoft.ServiceBus/namespaces/{namespaceName}/queues/{queueName}
	//  - /.../providers/Microsoft.ServiceBus/namespaces/{namespaceName}/topics/{topicName}/subscriptions/{subsName}
	if resID.ResourceProvider != resourceProviderServiceBus ||
		resID.Namespace == "" ||
		resID.ResourceType != resourceTypeQueues && resID.ResourceType != resourceTypeTopics ||
		resID.ResourceType == resourceTypeQueues && resID.SubResourceType != "" ||
		resID.ResourceType == resourceTypeTopics && resID.SubResourceType != resourceTypeSubscriptions {

		return nil, errors.New("resource ID does not refer to a Service Bus entity")
	}

	return resID, nil
}

// entityPath returns the entity path of the given Service Bus entity.
func entityPath(entityID *v1alpha1.AzureResourceID) string {
	switch entityID.ResourceType {
	case resourceTypeQueues:
		queueName := entityID.ResourceName
		return queueName
	case resourceTypeTopics:
		topicName := entityID.ResourceName
		subsName := entityID.SubResourceName
		return topicName + "/Subscriptions/" + subsName
	default:
		return ""
	}
}

// clientFromEnvironment mimics the behaviour of eventhub.NewHubFromEnvironment.
// It returns a azservicebus.Client that is suitable for the
// authentication method selected via environment variables.
func clientFromEnvironment(entityID *v1alpha1.AzureResourceID, clientOptions *azservicebus.ClientOptions) (*azservicebus.Client, error) {
	// SAS authentication (token, connection string)
	connStr := connectionStringFromEnvironment(entityID.Namespace, entityPath(entityID))
	if connStr != "" {
		client, err := azservicebus.NewClientFromConnectionString(connStr, clientOptions)
		if err != nil {
			return nil, fmt.Errorf("creating client from connection string: %w", err)
		}
		return client, nil
	}

	// AAD authentication (service principal)
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create Azure credentials: %w", err)
	}

	fqNamespace := entityID.Namespace + ".servicebus.windows.net"
	client, err := azservicebus.NewClient(fqNamespace, cred, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("creating client from service principal: %w", err)
	}
	return client, nil
}

// connectionStringFromEnvironment returns a Service Bus connection string
// based on values read from the environment.
func connectionStringFromEnvironment(namespace, entityPath string) string {
	connStr := os.Getenv(envConnStr)

	// if a key is set explicitly, it takes precedence and is used to
	// compose a new connection string
	if keyName, keyValue := os.Getenv(envKeyName), os.Getenv(envKeyValue); keyName != "" || keyValue != "" {
		azureEnv := &azure.PublicCloud
		connStr = fmt.Sprintf("Endpoint=sb://%s.%s;SharedAccessKeyName=%s;SharedAccessKey=%s;EntityPath=%s",
			namespace, azureEnv.ServiceBusEndpointSuffix, keyName, keyValue, entityPath)
	}

	return connStr
}

// Start implements adapter.Adapter.
//
// Required permissions:
//  Service Bus Queues:
//    - Microsoft.ServiceBus/namespaces/queues/read
//  Service Bus Topics:
//    - Microsoft.ServiceBus/namespaces/topics/read
//    - Microsoft.ServiceBus/namespaces/topics/subscriptions/read
//  Both (DataAction):
//  - Microsoft.ServiceBus/namespaces/messages/receive/action
func (a *adapter) Start(ctx context.Context) error {
	const maxMessages = 100
	logging.FromContext(ctx).Info("Listening for messages")

	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
			messages, err := a.msgRcvr.ReceiveMessages(ctx, maxMessages, nil)
			if err != nil {
				return fmt.Errorf("error receiving messages: %w", err)
			}
			for _, m := range messages {
				msg, err := toMessage(m)
				if err != nil {
					return fmt.Errorf("error transforming message: %w", err)
				}

				if err := a.handleMessage(ctx, msg); err != nil {
					return fmt.Errorf("error processing message: %w", err)
				}

				if err := a.msgRcvr.CompleteMessage(ctx, m, nil); err != nil {
					return fmt.Errorf("error completing message: %w", err)
				}
			}
		}
	}

	return nil
}

// handleMessage handles a single Service Bus message.
func (a *adapter) handleMessage(ctx context.Context, msg *Message) error {
	if msg == nil {
		return nil
	}

	events, err := a.msgPrcsr.Process(msg)
	if err != nil {
		return fmt.Errorf("processing Service Bus message with ID %s: %w", msg.ReceivedMessage.MessageID, err)
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

func webSocketsClientOptions(webSocketsEnable bool) *azservicebus.ClientOptions {
	clientOptions := &azservicebus.ClientOptions{}

	if webSocketsEnable {
		clientOptions = &azservicebus.ClientOptions{
			NewWebSocketConn: func(ctx context.Context, args azservicebus.NewWebSocketConnArgs) (net.Conn, error) {
				opts := &websocket.DialOptions{Subprotocols: []string{"amqp"}}
				wssConn, _, err := websocket.Dial(ctx, args.Host, opts)

				if err != nil {
					return nil, fmt.Errorf("creating client from connection string: %w", err)
				}

				return websocket.NetConn(context.Background(), wssConn, websocket.MessageBinary), nil
			},
		}
	}

	return clientOptions
}
