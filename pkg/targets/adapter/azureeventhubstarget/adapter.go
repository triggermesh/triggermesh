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

package azureeventhubstarget

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

const (
	resourceProviderEventHub = "Microsoft.EventHub"
	envKeyName               = "EVENTHUB_KEY_NAME"
	envKeyValue              = "EVENTHUB_KEY_VALUE"
	envConnStr               = "EVENTHUB_CONNECTION_STRING"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.AzureEventHubsTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeAzureEventHubsGenericResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	entityID, err := parseEventHubResourceID(env.HubResourceID)
	if err != nil {
		logger.Panicw("Unable to parse entity ID "+strconv.Quote(env.HubResourceID), zap.Error(err))
	}

	producerClient, err := clientFromEnvironment(entityID)
	if err != nil {
		logger.Panicw("Unable to create Event Hub client", zap.Error(err))
	}

	return &adapter{
		ehClient: producerClient,

		discardCEContext: env.DiscardCEContext,
		replier:          replier,
		ceClient:         ceClient,
		logger:           logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

type adapter struct {
	ehClient *azeventhubs.ProducerClient

	discardCEContext bool
	replier          *targetce.Replier
	ceClient         cloudevents.Client
	logger           *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Azure EventHub Target adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *adapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	if a.discardCEContext {
		batch, err := a.createEvent(ctx, event.Data())
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
		err = a.ehClient.SendEventDataBatch(ctx, batch, nil)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
	} else {
		// Serialize the event first, and then stream it
		bs, err := json.Marshal(event)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
		batch, err := a.createEvent(ctx, bs)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
		err = a.ehClient.SendEventDataBatch(ctx, batch, nil)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
	}

	return a.replier.Ok(&event, "ok")
}

func (a *adapter) createEvent(ctx context.Context, event []byte) (*azeventhubs.EventDataBatch, error) {
	batch, err := a.ehClient.NewEventDataBatch(ctx, &azeventhubs.EventDataBatchOptions{})
	if err != nil {
		return nil, err
	}

	err = batch.AddEventData(&azeventhubs.EventData{
		Body: event,
	}, nil)
	if err != nil {
		return nil, err
	}

	return batch, nil
}

// clientFromEnvironment returns a azeventhubs.ProducerClient that is suitable for the
// authentication method selected via environment variables.
func clientFromEnvironment(entityID *v1alpha1.AzureResourceID) (*azeventhubs.ProducerClient, error) {
	// SAS authentication (token, connection string)
	connStr := connectionStringFromEnvironment(entityID.Namespace, entityID.ResourceName)
	if connStr != "" {
		client, err := azeventhubs.NewProducerClientFromConnectionString(connStr, entityID.ResourceName, nil)
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
	client, err := azeventhubs.NewProducerClient(fqNamespace, entityID.ResourceName, cred, nil)
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
	if keyName, keyValue := os.Getenv(envKeyName), os.Getenv(envKeyValue); keyName != "" && keyValue != "" {
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

	return resID, nil
}
