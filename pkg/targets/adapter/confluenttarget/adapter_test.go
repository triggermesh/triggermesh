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
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/stretchr/testify/assert"
	zapt "go.uber.org/zap/zaptest"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	adaptertest "github.com/triggermesh/triggermesh/pkg/targets/adapter/test"
)

const (
	tEventID      = "123abv"
	tEventType    = "some.type"
	tEventSource  = "some.source"
	tEventSubject = "some subject"
	tEventData    = `{"hello":"world"}`
	tTopic        = "test-topic"
)

var (
	tkNoError      = kafka.NewError(kafka.ErrNoError, "", false)
	tkError        = kafka.NewError(kafka.ErrFail, "", false)
	tkUnknownTopic = kafka.NewError(kafka.ErrUnknownTopic, "", false)
)

func TestConfluentProduce(t *testing.T) {

	logger := zapt.NewLogger(t).Sugar()
	tc := map[string]struct {
		// produce data
		event        *cloudevents.Event
		produceError error

		// create topic data
		adminClient   KafkaAdminClient
		createTopic   bool
		topic         string
		existingTopic string

		// expectations
		expectedStartError string
		expectedResult     protocol.Result
	}{
		"produce event, error": {
			createTopic:  false,
			event:        createFakeEvent(),
			produceError: assert.AnError,

			expectedResult: cloudevents.ResultNACK,
		},

		"produce event, success": {
			createTopic:    false,
			event:          createFakeEvent(),
			expectedResult: cloudevents.ResultACK,
		},

		"ensuring topic, error retrieving metadata": {
			createTopic: true,
			topic:       tTopic,
			adminClient: newFakeAdminClient(withExistingMetadataTopic("", tkNoError, assert.AnError)),

			expectedStartError: assert.AnError.Error(),
		},

		"ensuring topic, metadata empty response": {
			createTopic: true,
			topic:       tTopic,
			adminClient: newFakeAdminClient(withExistingMetadataTopic("", tkNoError, nil)),

			expectedStartError: "empty response requesting topic metadata for",
		},

		"ensuring topic, metadata missing topic response": {
			createTopic: true,
			topic:       tTopic,
			adminClient: newFakeAdminClient(withExistingMetadataTopic(tTopic+"-no", tkNoError, nil)),

			expectedStartError: "metadata response does not contain required information",
		},

		"ensuring topic, topic exists": {
			createTopic: true,
			topic:       tTopic,
			adminClient: newFakeAdminClient(withExistingMetadataTopic(tTopic, tkNoError, nil)),
		},

		"ensuring topic, topic returned kafka error": {
			createTopic: true,
			topic:       tTopic,
			adminClient: newFakeAdminClient(withExistingMetadataTopic(tTopic, tkError, nil)),

			expectedStartError: "metadata returned inexpected status",
		},

		"creating topic, error on creation": {
			createTopic: true,
			topic:       tTopic,
			adminClient: newFakeAdminClient(
				withExistingMetadataTopic(tTopic, tkUnknownTopic, nil),
				withCreateTopic("", tkNoError, assert.AnError)),

			expectedStartError: "error creating topic",
		},

		"creating topic, returned kafka error": {
			createTopic: true,
			topic:       tTopic,
			adminClient: newFakeAdminClient(
				withExistingMetadataTopic(tTopic, tkUnknownTopic, nil),
				withCreateTopic(tTopic, tkError, nil)),

			expectedStartError: "failed to create topic",
		},

		"creating topic, success": {
			createTopic: true,
			topic:       tTopic,
			adminClient: newFakeAdminClient(
				withExistingMetadataTopic(tTopic, tkUnknownTopic, nil),
				withCreateTopic(tTopic, tkNoError, nil)),
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {

			ctx, cancel := context.WithCancel(context.Background())
			ceClient := &adaptertest.TestCloudEventsClient{}

			confluent := confluentAdapter{
				kafkaClient: newFakeKafkaClient(
					withProduceError(c.produceError),
					withAdmin(c.adminClient)),
				ceClient: ceClient,
				logger:   logger,
				topic:    c.topic,

				createTopicIfMissing: c.createTopic,
			}

			// start adapter
			end := make(chan error)
			go func() {
				err := confluent.Start(ctx)
				end <- err
			}()

			if c.event != nil {
				err := ceClient.WaitForReceiverReady(2 * time.Second)
				assert.NoError(t, err)

				res := ceClient.Send(ctx, *c.event)
				assert.Equalf(t, c.expectedResult, res, "returned wrong result from cloud event processing")
			}

			// stop adapter
			cancel()
			err := <-end
			if len(c.expectedStartError) == 0 {
				assert.NoError(t, err, "Target initialization or serving was unsuccessful")
			} else {
				if assert.NotNil(t, err, "expected error from adapter Start didnt occur") {
					assert.Contains(t, err.Error(), c.expectedStartError)
				}
			}
		})
	}
}

func createFakeEvent() *cloudevents.Event {
	event := cloudevents.NewEvent(cloudevents.VersionV1)

	event.SetID(tEventID)
	event.SetType(tEventType)
	event.SetSource(tEventSource)
	event.SetSubject(tEventSubject)
	if err := event.SetData(cloudevents.ApplicationJSON, tEventData); err != nil {
		panic(err)
	}

	return &event
}

// Mock kafka client

type fakeKOpts func(*fakeKafkaClient)

func newFakeKafkaClient(opts ...fakeKOpts) KafkaClient {
	fake := &fakeKafkaClient{}

	for _, f := range opts {
		f(fake)
	}
	return fake
}

type fakeKafkaClient struct {
	admin      KafkaAdminClient
	produceErr error
}

var _ KafkaClient = (*fakeKafkaClient)(nil)

func (c *fakeKafkaClient) Flush(timeoutMs int) int { return 0 }
func (c *fakeKafkaClient) Close()                  {}
func (c *fakeKafkaClient) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	deliveryChan <- msg
	return c.produceErr
}
func (c *fakeKafkaClient) CreateKafkaAdminClient() (KafkaAdminClient, error) {
	return c.admin, nil
}

func withProduceError(err error) fakeKOpts {
	return func(c *fakeKafkaClient) {
		c.produceErr = err
	}
}

func withAdmin(admin KafkaAdminClient) fakeKOpts {
	return func(c *fakeKafkaClient) {
		c.admin = admin
	}
}

// Mock kafka admin client

type fakeKAdminOpts func(*fakeKafkaAdminClient)

func newFakeAdminClient(opts ...fakeKAdminOpts) *fakeKafkaAdminClient {
	c := &fakeKafkaAdminClient{}
	for _, f := range opts {
		f(c)
	}
	return c
}

type fakeKafkaAdminClient struct {
	metadata      *kafka.Metadata
	metadataError error

	topicResult      []kafka.TopicResult
	createTopicError error
}

var _ KafkaAdminClient = (*fakeKafkaAdminClient)(nil)

func (c *fakeKafkaAdminClient) GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
	return c.metadata, c.metadataError
}

func (c *fakeKafkaAdminClient) CreateTopics(ctx context.Context, topics []kafka.TopicSpecification, options ...kafka.CreateTopicsAdminOption) (result []kafka.TopicResult, err error) {
	return c.topicResult, c.createTopicError
}

func (c *fakeKafkaAdminClient) Close() {
}

func withExistingMetadataTopic(topic string, kerr kafka.Error, err error) fakeKAdminOpts {
	return func(c *fakeKafkaAdminClient) {
		if topic != "" {
			c.metadata = &kafka.Metadata{
				Topics: map[string]kafka.TopicMetadata{
					topic: {
						Topic: topic,
						Error: kerr,
					},
				},
			}
		}
		c.metadataError = err
	}
}

func withCreateTopic(topic string, kerr kafka.Error, err error) fakeKAdminOpts {
	return func(c *fakeKafkaAdminClient) {
		if topic != "" {
			c.topicResult = []kafka.TopicResult{
				{
					Topic: topic,
					Error: kerr,
				},
			}
		}
		c.createTopicError = err
	}
}
