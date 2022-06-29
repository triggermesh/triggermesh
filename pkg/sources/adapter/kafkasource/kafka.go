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
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/Shopify/sarama"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

const (
	eventType = "io.triggermesh.kafka.event"
)

type consumerGroupHandler struct {
	adapter *kafkasourceAdapter
}

func newTLSCertificatesConfig(tlsConfig *tls.Config, clientCert, clientKey string) (*tls.Config, error) {
	if clientCert != "" && clientKey != "" {
		cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
		if err != nil {
			return tlsConfig, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

func newTLSRootCAConfig(tlsConfig *tls.Config, caCertFile string) *tls.Config {
	if caCertFile != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(caCertFile))
		tlsConfig.RootCAs = caCertPool
	}
	return tlsConfig
}

func (a *kafkasourceAdapter) emitEvent(ctx context.Context, msg sarama.ConsumerMessage) error {
	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(eventType)
	event.SetSubject("/kafka/target/event")
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
	ctx := context.Background()
	for msg := range claim.Messages() {
		err := c.adapter.emitEvent(ctx, *msg)
		if err != nil {
			c.adapter.logger.Errorf("Failed to emit event: %v", err)
		}
	}
	return nil
}

// TODO
func (consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error { return nil }

func (consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
