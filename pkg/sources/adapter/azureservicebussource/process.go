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

package azureservicebussource

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/Azure/azure-amqp-common-go/v3/uuid"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

// MessageProcessor converts an Service Bus message to a CloudEvent.
type MessageProcessor interface {
	Process(*Message) ([]*cloudevents.Event, error)
}

var _ MessageProcessor = (*defaultMessageProcessor)(nil)

// defaultMessageProcessor is the default processor for Service Bus messages.
type defaultMessageProcessor struct {
	ceSource string
}

// Process implements MessageProcessor.
func (p *defaultMessageProcessor) Process(msg *Message) ([]*cloudevents.Event, error) {
	event, err := makeServiceBusEvent(msg, p.ceSource)
	if err != nil {
		return nil, fmt.Errorf("creating CloudEvent from Service Bus message: %w", err)
	}

	return []*cloudevents.Event{event}, nil
}

// makeServiceBusEvent returns a CloudEvent for a generic Service Bus message.
func makeServiceBusEvent(msg *Message, srcAttr string) (*cloudevents.Event, error) {
	ceData := toCloudEventData(msg)

	event := cloudevents.NewEvent()
	event.SetID(msg.ReceivedMessage.MessageID)
	event.SetSource(srcAttr)
	event.SetType(v1alpha1.AzureEventType(sources.AzureServiceServiceBus, v1alpha1.AzureServiceBusGenericEventType))

	if msg.ScheduledEnqueueTime != nil {
		event.SetTime(*msg.ScheduledEnqueueTime)
	}

	if err := event.SetData(cloudevents.ApplicationJSON, ceData); err != nil {
		return nil, fmt.Errorf("setting CloudEvent data: %w", err)
	}

	return &event, nil
}

// toCloudEventData returns a servicebus.ReceivedMessage in a shape that is suitable for
// JSON serialization inside some CloudEvent data.
func toCloudEventData(msg *Message) interface{} {
	var data interface{}

	data = msg

	// if msg.Body contains raw JSON data, type it as json.RawMessage so
	// it doesn't get encoded to base64 during the serialization of the
	// CloudEvent data.
	var rawData json.RawMessage
	if err := json.Unmarshal(msg.Body, &rawData); err == nil {
		data = &MessageWithRawJSONData{
			Body:    rawData,
			Message: msg,
		}
	}

	return data
}

// Message is a servicebus.ReceivedMessage with some selected fields shadowed for
// improved serialization.
type Message struct {
	*azservicebus.ReceivedMessage
	LockToken *string
}

// MessageWithRawJSONData is an ReceivedMessage with RawMessage-typed JSON data.
type MessageWithRawJSONData struct {
	Body json.RawMessage
	*Message
}

// toMessage converts a azservicebus.ReceivedMessage into a Message
// removing a new parameter (RawAMQPMessage) introduced in azservicebus v1.1.0
// that breaks our serialization.
func toMessage(rcvMsg *azservicebus.ReceivedMessage) (*Message, error) {
	return &Message{
		ReceivedMessage: &azservicebus.ReceivedMessage{
			ApplicationProperties:      rcvMsg.ApplicationProperties,
			Body:                       rcvMsg.Body,
			ContentType:                rcvMsg.ContentType,
			CorrelationID:              rcvMsg.CorrelationID,
			DeadLetterErrorDescription: rcvMsg.DeadLetterErrorDescription,
			DeadLetterReason:           rcvMsg.DeadLetterReason,
			DeadLetterSource:           rcvMsg.DeadLetterSource,
			DeliveryCount:              rcvMsg.DeliveryCount,
			EnqueuedSequenceNumber:     rcvMsg.EnqueuedSequenceNumber,
			EnqueuedTime:               rcvMsg.EnqueuedTime,
			ExpiresAt:                  rcvMsg.ExpiresAt,
			LockedUntil:                rcvMsg.LockedUntil,
			LockToken:                  rcvMsg.LockToken,
			MessageID:                  rcvMsg.MessageID,
			PartitionKey:               rcvMsg.PartitionKey,
			ReplyTo:                    rcvMsg.ReplyTo,
			ReplyToSessionID:           rcvMsg.ReplyToSessionID,
			ScheduledEnqueueTime:       rcvMsg.ScheduledEnqueueTime,
			SequenceNumber:             rcvMsg.SequenceNumber,
			SessionID:                  rcvMsg.SessionID,
			State:                      rcvMsg.State,
			Subject:                    rcvMsg.Subject,
			TimeToLive:                 rcvMsg.TimeToLive,
			To:                         rcvMsg.To,
		},
		LockToken: stringifyLockToken((*uuid.UUID)(&rcvMsg.LockToken)),
	}, nil
}

// stringifyLockToken converts a UUID byte-array into its string representation.
func stringifyLockToken(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}

	return to.Ptr(id.String())
}
