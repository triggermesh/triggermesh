/*
Copyright (c) 2021 TriggerMesh Inc.

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

package confluenttarget

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
	"knative.dev/pkg/logging"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)

	logger := logging.FromContext(ctx)

	kafkaClient, err := NewKafkaClient(&kafka.ConfigMap{
		"bootstrap.servers":       env.BootstrapServers,
		"sasl.username":           env.SASLUsername,
		"sasl.password":           env.SASLPassword,
		"sasl.mechanisms":         env.SASLMechanisms,
		"security.protocol":       env.SecurityProtocol,
		"broker.version.fallback": env.BrokerVersionFallback,
		"api.version.fallback.ms": env.APIVersionFallbackMs,
	})
	if err != nil {
		logger.Panic(err)
	}

	return &confluentAdapter{
		kafkaClient:               kafkaClient,
		topic:                     env.Topic,
		createTopicIfMissing:      env.CreateTopicIfMissing,
		flushTimeout:              env.FlushOnExitTimeoutMillisecs,
		topicTimeout:              env.CreateTopicTimeoutMillisecs,
		newTopicPartitions:        env.NewTopicPartitions,
		newTopicReplicationFactor: env.NewTopicReplicationFactor,

		discardCEContext: env.DiscardCEContext,

		ceClient: ceClient,
		logger:   logger,
	}
}

var _ pkgadapter.Adapter = (*confluentAdapter)(nil)

type confluentAdapter struct {
	kafkaClient KafkaClient
	topic       string

	createTopicIfMissing bool

	flushTimeout              int
	topicTimeout              int
	newTopicPartitions        int
	newTopicReplicationFactor int

	discardCEContext bool

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Returns if stopCh is closed or Send() returns an error.
func (a *confluentAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Confluent adapter")

	defer func() {
		a.kafkaClient.Flush(a.flushTimeout)
		a.kafkaClient.Close()
	}()

	if a.createTopicIfMissing {
		if err := a.ensureTopic(ctx, a.topic); err != nil {
			return fmt.Errorf("failed ensuring Topic %s: %w", a.topic, err)
		}
	}

	if err := a.ceClient.StartReceiver(ctx, a.dispatch); err != nil {
		return fmt.Errorf("error starting the cloud events server: %w", err)
	}
	return nil
}

func (a *confluentAdapter) dispatch(event cloudevents.Event) cloudevents.Result {
	var msgVal []byte

	if a.discardCEContext {
		msgVal = event.Data()
	} else {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			a.logger.Errorw("Error marshalling CloudEvent", zap.Error(err))
			return cloudevents.ResultNACK
		}
		msgVal = jsonEvent
	}

	km := &kafka.Message{
		Key:            []byte(event.ID()),
		TopicPartition: kafka.TopicPartition{Topic: &a.topic, Partition: kafka.PartitionAny},
		Value:          msgVal,
	}

	// librdkafka provides buffering, we set channel size to 1
	// to avoid blocking tests as they execute in the same thread
	deliveryChan := make(chan kafka.Event, 1)
	defer close(deliveryChan)

	if err := a.kafkaClient.Produce(km, deliveryChan); err != nil {
		a.logger.Errorw("Error producing Kafka message", zap.String("msg", km.String()), zap.Error(err))
		return cloudevents.ResultNACK
	}

	r := <-deliveryChan
	m := r.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		a.logger.Infof("Message delivery failed: %v", m.TopicPartition.Error)
		return cloudevents.ResultNACK
	}

	a.logger.Debugf("Delivered message to topic %s [%d] at offset %v",
		*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)

	return cloudevents.ResultACK
}

//ensureTopic creates a topic if missing
func (a *confluentAdapter) ensureTopic(ctx context.Context, topic string) error {
	a.logger.Infof("Ensuring topic %q", topic)

	adminClient, err := a.kafkaClient.CreateKafkaAdminClient()
	if err != nil {
		return fmt.Errorf("error creating admin client from producer: %w", err)
	}
	defer adminClient.Close()

	ts := []kafka.TopicSpecification{{
		Topic:             topic,
		NumPartitions:     a.newTopicPartitions,
		ReplicationFactor: a.newTopicReplicationFactor}}

	m, err := adminClient.GetMetadata(&topic, false, a.topicTimeout)
	if err != nil {
		return fmt.Errorf("error retrieving topic %q metadata: %w", topic, err)
	}
	if m == nil {
		return fmt.Errorf("empty response requesting topic metadata for %q", a.topic)
	}

	t, ok := m.Topics[a.topic]
	if !ok {
		return fmt.Errorf("topic %q metadata response does not contain required information", a.topic)
	}

	switch t.Error.Code() {
	case kafka.ErrNoError:
		a.logger.Infof("Topic found: %q with %d partitions", t.Topic, len(t.Partitions))
		return nil
	case kafka.ErrUnknownTopic, kafka.ErrUnknownTopicOrPart:
		// topic does not exists, we need to create it.
	default:
		return fmt.Errorf("topic %q metadata returned inexpected status: %w", a.topic, t.Error)
	}

	a.logger.Infof("Creating topic %q", topic)
	results, err := adminClient.CreateTopics(ctx, ts, kafka.SetAdminOperationTimeout(time.Duration(a.topicTimeout)*time.Millisecond))
	if err != nil {
		return fmt.Errorf("error creating topic %q: %w", a.topic, err)
	}

	if len(results) != 1 {
		return fmt.Errorf("creating topic %s returned inexpected results: %+v", a.topic, results)
	}

	if results[0].Error.Code() != kafka.ErrNoError {
		return fmt.Errorf("failed to create topic %s: %w", a.topic, results[0].Error)
	}

	return nil
}
