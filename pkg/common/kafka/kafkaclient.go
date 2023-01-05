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

package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

// SaramaCachedClient wraps a sarama kafka client, exposing
// common used methods at sources and targets.
//
// If a method fails it will try to re-connect and try again
// before failing.
type SaramaCachedClient struct {
	bootstrapServers []string
	cfg              *sarama.Config

	client       sarama.Client
	admin        sarama.ClusterAdmin
	syncProducer sarama.SyncProducer

	refresh *time.Duration

	m sync.RWMutex
}

type SaramaCachedClientOption func(*SaramaCachedClient)

func WithSaramaCachedClientRefresh(d time.Duration) SaramaCachedClientOption {
	return func(scc *SaramaCachedClient) {
		scc.refresh = &d
	}
}

func NewSaramaCachedClient(ctx context.Context, bootstrapServers []string, cfg *sarama.Config, logger *zap.Logger, opts ...SaramaCachedClientOption) (*SaramaCachedClient, error) {
	sarama.Logger = zap.NewStdLog(logger)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("sarama kafka client configuration is not valid: %w", err)
	}

	scc := &SaramaCachedClient{
		bootstrapServers: bootstrapServers,
		cfg:              cfg,
	}
	for _, opt := range opts {
		opt(scc)
	}

	// Initialize
	if err := scc.RefreshProducerClients(); err != nil {
		return nil, fmt.Errorf("could not create kafka client: %w", err)
	}

	// All methods that operate on kafka have an error management pattern that will
	// re-create connections and try again. The refresh routine adds a more pro-active
	// approach, re-creating connections when they reach the refresh period.
	if scc.refresh != nil {
		go func() {
			for {
				select {
				case <-time.After(*scc.refresh):
					if err := scc.RefreshProducerClients(); err != nil {
						logger.Error("Could not refresh kafka clients", zap.Error(err))
					}

				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return scc, nil
}

func (sc *SaramaCachedClient) Close() error {
	sc.m.Lock()
	defer sc.m.Unlock()

	if sc.client == nil {
		// no client connection is expected to exist.
		return nil
	}

	var err error
	if e := sc.syncProducer.Close(); e != nil {
		err = e
	}

	if e := sc.admin.Close(); e != nil {
		if err == nil {
			err = e
		} else {
			err = fmt.Errorf("%w; %w", err, e)
		}
	}

	if e := sc.client.Close(); e != nil {
		if err == nil {
			err = e
		} else {
			err = fmt.Errorf("%w; %w", err, e)
		}
	}

	return err
}

func (sc *SaramaCachedClient) RefreshProducerClients() error {
	sc.m.Lock()
	defer sc.m.Unlock()

	client, err := sarama.NewClient(
		sc.bootstrapServers,
		sc.cfg,
	)
	if err != nil {
		return fmt.Errorf("could not create sarama kafka client: %w", err)
	}

	// ignore errors since we are discarding the existing connection.
	if sc.client != nil {
		_ = sc.client.Close()
	}
	if sc.admin != nil {
		_ = sc.admin.Close()
	}
	if sc.syncProducer != nil {
		_ = sc.syncProducer.Close()
	}

	sc.client = client

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		return fmt.Errorf("could not create sarama kafka admin client: %w", err)
	}

	sc.admin = admin

	sp, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return fmt.Errorf("could not create sarama kafka sync producer client: %w", err)
	}

	sc.syncProducer = sp

	return nil
}

func (sc *SaramaCachedClient) ensureTopic(topic string, replicas int16, partitions int32) error {
	topicDetail := &sarama.TopicDetail{
		ReplicationFactor: replicas,
		NumPartitions:     partitions,
	}

	sc.m.RLock()
	defer sc.m.RUnlock()

	err := sc.admin.CreateTopic(topic, topicDetail, false)
	if err, ok := err.(*sarama.TopicError); ok && err.Err != sarama.ErrTopicAlreadyExists {
		return err
	}

	return nil
}

func (sc *SaramaCachedClient) EnsureTopic(topic string, replicas int16, partitions int32) error {
	if err := sc.ensureTopic(topic, replicas, partitions); err == nil {
		return nil
	}

	// if there is an error, try to re-create the producer clients
	if err := sc.RefreshProducerClients(); err != nil {
		return err
	}

	return sc.ensureTopic(topic, replicas, partitions)
}

func (sc *SaramaCachedClient) pingTopic(topic string) error {
	sc.m.RLock()
	defer sc.m.RUnlock()

	_, err := sc.admin.DescribeTopics([]string{topic})
	if err, ok := err.(*sarama.TopicError); ok && err.Err != sarama.ErrTopicAlreadyExists {
		return err
	}

	return nil
}

func (sc *SaramaCachedClient) PingTopic(topic string) error {
	if err := sc.pingTopic(topic); err == nil {
		return nil
	}

	// if there is an error, try to re-create the producer clients
	if err := sc.RefreshProducerClients(); err != nil {
		return err
	}

	return sc.pingTopic(topic)
}

func (sc *SaramaCachedClient) sendMessageSync(message *sarama.ProducerMessage) error {
	_, _, err := sc.syncProducer.SendMessage(message)
	return err
}

func (sc *SaramaCachedClient) SendMessageSync(message *sarama.ProducerMessage) error {
	if err := sc.sendMessageSync(message); err == nil {
		return nil
	}

	// if there is an error, try to re-create the producer clients
	if err := sc.RefreshProducerClients(); err != nil {
		return err
	}

	return sc.sendMessageSync(message)
}
