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
	"log"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	servicebus "github.com/Azure/azure-service-bus-go"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig
	ConnectionString string `envconfig:"SERVICEBUS_CONNECTION_STRING" required:"true"`
	Subscription     string `envconfig:"SERVICEBUS_SUBSCRIPTION" required:"true"`
}

// adapter implements the source's adapter.
type adapter struct {
	sub    *servicebus.Subscription
	source string

	logger   *zap.SugaredLogger
	ceClient cloudevents.Client
}

// MessageWithRawData is an *servicebus.Message with RawMessage-typed data.
type MessageWithRawData struct {
	Data json.RawMessage
	*servicebus.Message
}

var _ pkgadapter.Adapter = (*adapter)(nil)

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)
	env := envAcc.(*envConfig)
	var tn string

	s := strings.Split(env.ConnectionString, ";")
	entityPath := strings.Split(s[3], "=")
	tn = entityPath[1]
	ns, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(env.ConnectionString))
	if err != nil {
		log.Fatal(err)
		return nil
	}

	tm := ns.NewTopicManager()
	te, err := tm.Get(ctx, tn)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	topic, err := ns.NewTopic(te.Name)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	sub, err := topic.NewSubscription(env.Subscription)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	source := v1alpha1.AzureServiceBusTopicSourceName(env.Namespace, env.Name)
	return &adapter{
		sub:    sub,
		source: source,

		logger:   logger,
		ceClient: ceClient,
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	var printMessage servicebus.HandlerFunc = func(ctx context.Context, msg *servicebus.Message) error {
		if err := a.handleMessage(ctx, msg); err != nil {
			return err
		}
		if err := msg.Complete(ctx); err != nil {
			a.logger.Error(err)
			return err
		}
		return nil
	}

	if err := a.sub.Receive(ctx, printMessage); err != nil {
		a.logger.Error(err)
		return err
	}

	return nil
}

func (a *adapter) handleMessage(ctx context.Context, msg *servicebus.Message) error {
	ced := toCloudEventData(msg)
	if err := a.sendCloudEvent(ced); err != nil {
		a.logger.Error(err)
		return err
	}

	return nil
}

func (a *adapter) sendCloudEvent(m interface{}) error {
	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(v1alpha1.AzureEventType(sources.AzureServiceBusTopic, v1alpha1.AzureServiceBusbGenericEventType))
	event.SetSource(a.source)
	if err := event.SetData(cloudevents.ApplicationJSON, m); err != nil {
		return fmt.Errorf("setting event data: %w", err)
	}

	if result := a.ceClient.Send(context.Background(), event); !cloudevents.IsACK(result) {
		return fmt.Errorf("sending CloudEvent: %w", result)
	}

	return nil
}

func toCloudEventData(e *servicebus.Message) interface{} {
	var data interface{}
	data = e

	// if event.Data contains raw JSON data, type it as json.RawMessage so
	// it doesn't get encoded to base64 during the serialization of the
	// CloudEvent data.
	var rawData json.RawMessage
	if err := json.Unmarshal(e.Data, &rawData); err == nil {
		data = MessageWithRawData{
			Data:    rawData,
			Message: e,
		}
	}

	return data
}
