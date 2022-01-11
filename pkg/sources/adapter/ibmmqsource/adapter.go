//go:build !codeanalysis

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

package ibmmqsource

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/ibmmqsource/mq"
)

var _ pkgadapter.Adapter = (*ibmmqsourceAdapter)(nil)

type ibmmqsourceAdapter struct {
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	mqEnvs *SourceEnvAccessor
}

// NewAdapter returns adapter implementation
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*SourceEnvAccessor)
	logger := logging.FromContext(ctx)

	return &ibmmqsourceAdapter{
		ceClient: ceClient,
		logger:   logger,
		mqEnvs:   env,
	}
}

// Returns if stopCh is closed or Send() returns an error.
func (a *ibmmqsourceAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting IBMMQSource Adapter")

	conn, err := mq.NewConnection(a.mqEnvs.EnvConnectionConfig.ConnectionConfig())
	if err != nil {
		return fmt.Errorf("failed to create IBM MQ connection: %w", err)
	}
	defer conn.Disc()

	queue, err := mq.OpenQueue(a.mqEnvs.EnvConnectionConfig.QueueName, conn)
	if err != nil {
		return fmt.Errorf("failed to open IBM MQ queue: %w", err)
	}
	defer queue.Close()

	err = queue.RegisterCallback(a.eventHandler(), a.mqEnvs.Delivery(), a.logger)
	if err != nil {
		return fmt.Errorf("failed to register callback: %w", err)
	}
	defer queue.DeleteMessageHandle()
	defer queue.DeregisterCallback()

	if err := queue.StartListen(conn); err != nil {
		return fmt.Errorf("failed to start callback listener: %w", err)
	}
	defer queue.StopCallback(conn)

	<-ctx.Done()
	return nil
}

func (a *ibmmqsourceAdapter) eventHandler() mq.Handler {
	return func(data []byte, correlID string) error {
		event := cloudevents.NewEvent(cloudevents.VersionV1)
		event.SetType(v1alpha1.IBMMQSourceEventType)
		event.SetSource(fmt.Sprintf("%s/%s", a.mqEnvs.EnvConnectionConfig.ConnectionName, strings.ToLower(a.mqEnvs.EnvConnectionConfig.ChannelName)))
		if correlID != "" {
			event.SetExtension(mq.CECorrelIDAttr, correlID)
		}
		if err := event.SetData(cloudevents.ApplicationJSON, data); err != nil {
			a.logger.Errorf("Can't set Cloudevent data: %v", err)
			return err
		}
		if res := a.ceClient.Send(context.Background(), event); cloudevents.IsUndelivered(res) {
			a.logger.Errorf("Cloudevent is not delivered: %v\n", res)
			return res
		}
		return nil
	}
}
