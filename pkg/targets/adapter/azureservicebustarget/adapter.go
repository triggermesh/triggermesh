/*
Copyright 2023 TriggerMesh Inc.

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

package azureservicebustarget

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"nhooyr.io/websocket"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
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

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.AzureServiceBusTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeAzureServiceBusGenericResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	entityID, err := parseServiceBusResourceID(env.EntityResourceID)
	if err != nil {
		logger.Panicw("Unable to parse entity ID "+strconv.Quote(env.EntityResourceID), zap.Error(err))
	}

	client, err := clientFromEnvironment(entityID, newAzureServiceBusClientOptions(
		webSocketsClientOption(env.WebSocketsEnable)))
	if err != nil {
		logger.Panicw("Unable to obtain interface for Service Bus Namespace", zap.Error(err))
	}

	sender, err := client.NewSender(entityID.ResourceName, nil)
	if err != nil {
		logger.Panicw("Unable to obtain sender for Service Bus Namespace", zap.Error(err))
	}

	return &adapter{
		sender: sender,

		discardCEContext: env.DiscardCEContext,
		replier:          replier,
		ceClient:         ceClient,
		logger:           logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

type adapter struct {
	sender *azservicebus.Sender

	discardCEContext bool
	replier          *targetce.Replier
	ceClient         cloudevents.Client
	logger           *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Azure Service Bus Target adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *adapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	ceTypeTag := metrics.TagEventType(event.Type())
	ceSrcTag := metrics.TagEventSource(event.Source())

	start := time.Now()
	defer func() {
		a.sr.ReportProcessingLatency(time.Since(start), ceTypeTag, ceSrcTag)
	}()

	var msg []byte
	if a.discardCEContext {
		msg = event.Data()
	} else {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			a.logger.Errorw("Error marshalling CloudEvent", zap.Error(err))
			a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
		msg = jsonEvent
	}
	if err := a.sender.SendMessage(ctx, &azservicebus.Message{Body: msg}, nil);  err != nil {
		a.logger.Errorw("Error sending message to Service Bus", zap.Error(err))
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	return a.replier.Ok(&event, "ok")
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
		resID.ResourceType != resourceTypeQueues && resID.ResourceType != resourceTypeTopics {
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

// clientFromEnvironment mimics the behaviour of servicebus.NewHubFromEnvironment.
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

type clientOption func(*azservicebus.ClientOptions)

func newAzureServiceBusClientOptions(opts ...clientOption) *azservicebus.ClientOptions {
	co := &azservicebus.ClientOptions{}
	for _, opt := range opts {
		opt(co)
	}
	return co
}

func webSocketsClientOption(webSocketsEnable bool) clientOption {
	return func(opts *azservicebus.ClientOptions) {

		if webSocketsEnable {
			opts.NewWebSocketConn = func(ctx context.Context, args azservicebus.NewWebSocketConnArgs) (net.Conn, error) {
				opts := &websocket.DialOptions{Subprotocols: []string{"amqp"}}
				wssConn, _, err := websocket.Dial(ctx, args.Host, opts)

				if err != nil {
					return nil, fmt.Errorf("creating client: %w", err)
				}

				return websocket.NetConn(context.Background(), wssConn, websocket.MessageBinary), nil
			}
		}
	}
}
