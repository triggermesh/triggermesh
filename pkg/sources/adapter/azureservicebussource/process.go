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

package azureservicebussource

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/Azure/azure-amqp-common-go/v3/uuid"
	servicebus "github.com/Azure/azure-service-bus-go"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

// MessageProcessor converts an Service Bus message to a CloudEvent.
type MessageProcessor interface {
	Process(*servicebus.Message) ([]*cloudevents.Event, error)
}

var _ MessageProcessor = (*defaultMessageProcessor)(nil)

// defaultMessageProcessor is the default processor for Service Bus messages.
type defaultMessageProcessor struct {
	ceSource string
}

// Process implements MessageProcessor.
func (p *defaultMessageProcessor) Process(msg *servicebus.Message) ([]*cloudevents.Event, error) {
	event, err := makeServiceBusEvent(msg, p.ceSource)
	if err != nil {
		return nil, fmt.Errorf("creating CloudEvent from Service Bus message: %w", err)
	}

	return []*cloudevents.Event{event}, nil
}

// makeServiceBusEvent returns a CloudEvent for a generic Service Bus message.
func makeServiceBusEvent(msg *servicebus.Message, srcAttr string) (*cloudevents.Event, error) {
	ceData := toCloudEventData(msg)

	event := cloudevents.NewEvent()
	event.SetID(msg.ID)
	event.SetSource(srcAttr)
	event.SetType(v1alpha1.AzureEventType(sources.AzureServiceServiceBus, v1alpha1.AzureServiceBusGenericEventType))

	if sysProps := msg.SystemProperties; sysProps != nil && sysProps.EnqueuedTime != nil {
		event.SetTime(*sysProps.EnqueuedTime)
	}

	if err := event.SetData(cloudevents.ApplicationJSON, ceData); err != nil {
		return nil, fmt.Errorf("setting CloudEvent data: %w", err)
	}

	return &event, nil
}

// toCloudEventData returns a servicebus.Message in a shape that is suitable for
// JSON serialization inside some CloudEvent data.
func toCloudEventData(msg *servicebus.Message) interface{} {
	var data interface{}

	serialMsg := &Message{
		LockToken: stringifyLockToken(msg.LockToken),
		Message:   msg,
	}

	data = serialMsg

	// if event.Data contains raw JSON data, type it as json.RawMessage so
	// it doesn't get encoded to base64 during the serialization of the
	// CloudEvent data.
	var rawData json.RawMessage
	if err := json.Unmarshal(msg.Data, &rawData); err == nil {
		data = &MessageWithRawJSONData{
			Data:    rawData,
			Message: serialMsg,
		}
	}

	return data
}

// Message is a servicebus.Message with some selected fields shadowed for
// improved serialization.
type Message struct {
	LockToken *string
	*servicebus.Message
}

// MessageWithRawJSONData is an Message with RawMessage-typed JSON data.
type MessageWithRawJSONData struct {
	Data json.RawMessage
	*Message
}

// stringifyLockToken converts a UUID byte-array into its string representation.
func stringifyLockToken(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}

	return to.StringPtr(id.String())
}
