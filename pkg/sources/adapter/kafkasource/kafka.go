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

package kafkasource

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
)

const (
	eventType = "io.triggermesh.kafka.event"
)

type consumerGroupHandler struct {
	adapter *kafkasourceAdapter
}

func (a *kafkasourceAdapter) emitEvent(ctx context.Context, msg sarama.ConsumerMessage) error {
	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(eventType)
	event.SetSubject("kafka/event")
	event.SetSource(msg.Topic)
	event.SetID(string(msg.Key))

	if err := event.SetData(cloudevents.ApplicationJSON, msg.Value); err != nil {
		return fmt.Errorf("failed to set event data: %w", err)
	}

	if result := a.ceClient.Send(context.Background(), event); !cloudevents.IsACK(result) {
		return result
	}
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if err := c.adapter.emitEvent(session.Context(), *msg); err != nil {
				c.adapter.logger.Errorw("Failed to emit event: %v", zap.Error(err))
				// do not mark message
				continue
			}
			session.MarkMessage(msg, "")

		case <-session.Context().Done():
			c.adapter.logger.Infow("Context closed, exiting consumer")
			return nil
		}
	}
}

func (c consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}
