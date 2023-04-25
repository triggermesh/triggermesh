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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/devigned/tab"
	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/protocol"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/go-autorest/autorest/azure"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/azureeventhubssource/trace"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/health"
)

const (
	resourceProviderEventHub = "Microsoft.EventHub"
	envKeyName               = "EVENTHUB_KEY_NAME"
	envKeyValue              = "EVENTHUB_KEY_VALUE"
	envConnStr               = "EVENTHUB_CONNECTION_STRING"

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

	runtimeInfo *azeventhubs.EventHubProperties
	ehClient    *azeventhubs.ConsumerClient
	ceClient    cloudevents.Client

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

	entityID, err := parseEventHubResourceID(env.HubResourceID)
	if err != nil {
		logger.Panicw("Unable to parse entity ID "+strconv.Quote(env.HubResourceID), zap.Error(err))
	}

	consumerClient, err := clientFromEnvironment(entityID)
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

	consumerGroup := azeventhubs.DefaultConsumerGroup
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
		ehClient: consumerClient,

		ehConsumerGroup: consumerGroup,

		msgPrcsr: msgPrcsr,
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	defer func() {
		err := a.ehClient.Close(ctx)
		if err != nil {
			a.logger.Errorw("Unable to close Event Hub client", zap.Error(err))
		}
	}()

	runtimeInfo, err := a.ehClient.GetEventHubProperties(ctx, nil)
	if err != nil {
		return fmt.Errorf("getting Event Hub runtime information: %w", err)
	}
	a.runtimeInfo = &runtimeInfo

	// TODO(antoineco): Find a way to inject Prometheus metric tags into
	// the context.Context that is passed to handleMessage().
	// Currently, the SDK always passes context.Background(), instead of our ctx:
	// https://github.com/Azure/azure-event-hubs-go/blob/v3.3.17/receiver.go#L219
	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	// listen to each partition of the Event Hub
	wg := sync.WaitGroup{}
	a.logger.Info("Starting Event Hub message receivers for partitions ", runtimeInfo.PartitionIDs)
	for _, partition := range a.runtimeInfo.PartitionIDs {
		wg.Add(1)
		go func(partitionID string) {
			defer wg.Done()
			a.processPartition(ctx, partitionID)
		}(partition)
	}
	health.MarkReady()
	wg.Wait()
	return nil
}

// processPartition processes events from a single partition of the Event Hub.
func (a *adapter) processPartition(ctx context.Context, partitionID string) {
	partitionClient, err := a.ehClient.NewPartitionClient(partitionID, &azeventhubs.PartitionClientOptions{
		StartPosition: azeventhubs.StartPosition{
			Latest: to.Ptr(true),
		},
	})
	if err != nil {
		a.logger.Errorf("creating partition client for partition %s: %v", partitionID, err)
		return
	}
	defer partitionClient.Close(ctx)

	for {
		select {
		case <-ctx.Done():
			continue
		default:
			receiveCtx, cancel := context.WithTimeout(ctx, connTimeout)

			events, err := partitionClient.ReceiveEvents(receiveCtx, 100, nil)
			cancel()

			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				if ctx.Err() != nil {
					a.logger.Info("[Partition: %s] Application is stopping, stopping receive for partition\n", partitionID)
					break
				}
			} else if err != nil {
				a.logger.Errorf("receiving events for partition %s: %v", partitionID, err)
			}

			for _, event := range events {
				err = a.handleMessage(ctx, event)
				if err != nil {
					a.logger.Errorw("Error handling message", zap.Error(err))
				}
			}
		}
	}
}

// handleMessage satisfies eventhub.Handler.
func (a *adapter) handleMessage(ctx context.Context, msg *azeventhubs.ReceivedEventData) error {
	if msg == nil {
		return nil
	}

	events, err := a.msgPrcsr.Process(msg)
	if err != nil {
		return fmt.Errorf("processing Event Hubs message with ID %s: %w", *msg.MessageID, err)
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

// clientFromEnvironment returns a azeventhubs.ConsumerClient that is suitable for the
// authentication method selected via environment variables.
func clientFromEnvironment(entityID *v1alpha1.AzureResourceID) (*azeventhubs.ConsumerClient, error) {
	// SAS authentication (token, connection string)
	connStr := connectionStringFromEnvironment(entityID.Namespace, entityID.ResourceName)
	if connStr != "" {
		client, err := azeventhubs.NewConsumerClientFromConnectionString(connStr, entityID.ResourceName, azeventhubs.DefaultConsumerGroup, nil)
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
	client, err := azeventhubs.NewConsumerClient(fqNamespace, entityID.ResourceName, azeventhubs.DefaultConsumerGroup, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating client from service principal: %w", err)
	}
	return client, nil
}

// connectionStringFromEnvironment returns a EventHub connection string
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

// parseEventHubResourceID parses the given resource ID string to a
// structured resource ID, and validates that this resource ID refers to a
// EventHub entity.
func parseEventHubResourceID(resIDStr string) (*v1alpha1.AzureResourceID, error) {
	resID := &v1alpha1.AzureResourceID{}

	err := json.Unmarshal([]byte(strconv.Quote(resIDStr)), resID)
	if err != nil {
		return nil, fmt.Errorf("deserializing resource ID string: %w", err)
	}

	// Must match the following pattern:
	//  - /.../providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventhub}
	if resID.ResourceProvider != resourceProviderEventHub || resID.Namespace == "" {
		return nil, errors.New("resource ID does not refer to a Event Hub entity")
	}

	return resID, nil
}
