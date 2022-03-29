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

package confluenttarget

import (
	"context"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

// KafkaAdminClient is a wrapper of the Confluent Kafka admin
// functions needed for the Confluent adapter.
type KafkaAdminClient interface {
	GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)
	CreateTopics(ctx context.Context, topics []kafka.TopicSpecification, options ...kafka.CreateTopicsAdminOption) (result []kafka.TopicResult, err error)
	Close()
}

// KafkaClient is a wrapper of the Confluent Kafka producer
// functions needed for the Confluent adapter.
type KafkaClient interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	CreateKafkaAdminClient() (KafkaAdminClient, error)
	Flush(timeoutMs int) int
	Close()
}

// NewKafkaClient creates a Kafka client
func NewKafkaClient(cfg *kafka.ConfigMap) (KafkaClient, error) {
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		return nil, err
	}

	return &kafkaClient{producer: producer}, nil
}

type kafkaClient struct {
	producer *kafka.Producer
}

// Flush and wait for outstanding messages and requests to complete delivery.
func (c *kafkaClient) Flush(timeoutMs int) int {
	return c.producer.Flush(timeoutMs)
}

// Close client connection
func (c *kafkaClient) Close() {
	c.producer.Close()
}

// Produce single message.
func (c *kafkaClient) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	return c.producer.Produce(msg, deliveryChan)
}

// CreateKafkaAdminClient creates the default implementation of KafkaAdminClient
func (c *kafkaClient) CreateKafkaAdminClient() (KafkaAdminClient, error) {
	adminClient, err := kafka.NewAdminClientFromProducer(c.producer)
	if err != nil {
		return nil, err
	}

	return &kafkaAdminClient{adminClient: adminClient}, nil
}

type kafkaAdminClient struct {
	adminClient *kafka.AdminClient
}

// GetMetadata queries for cluster and topic metadata.
func (c *kafkaAdminClient) GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
	return c.adminClient.GetMetadata(topic, allTopics, timeoutMs)
}

// CreateTopics creates topics in cluster.
func (c *kafkaAdminClient) CreateTopics(ctx context.Context, topics []kafka.TopicSpecification, options ...kafka.CreateTopicsAdminOption) (result []kafka.TopicResult, err error) {
	return c.adminClient.CreateTopics(ctx, topics, options...)
}

// Close admin client
func (c *kafkaAdminClient) Close() {
	c.adminClient.Close()
}
