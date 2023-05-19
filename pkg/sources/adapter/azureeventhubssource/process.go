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

package azureeventhubssource

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
)

// MessageProcessor converts an Event Hubs message to a CloudEvent.
type MessageProcessor interface {
	Process(*azeventhubs.ReceivedEventData) ([]*cloudevents.Event, error)
}

var (
	_ MessageProcessor = (*defaultMessageProcessor)(nil)
	_ MessageProcessor = (*eventGridMessageProcessor)(nil)
)

// defaultMessageProcessor is the default processor for Event Hubs messages.
type defaultMessageProcessor struct {
	ceSource string
	ceType   string
}

// Process implements MessageProcessor.
func (p *defaultMessageProcessor) Process(msg *azeventhubs.ReceivedEventData) ([]*cloudevents.Event, error) {
	event, err := makeEventHubsEvent(msg, p.ceSource, p.ceType)
	if err != nil {
		return nil, fmt.Errorf("creating CloudEvent from Event Hubs message: %w", err)
	}

	return []*cloudevents.Event{event}, nil
}

// makeEventHubsEvent returns a CloudEvent for a generic Event Hubs message.
func makeEventHubsEvent(msg *azeventhubs.ReceivedEventData, srcAttr, typeAttr string) (*cloudevents.Event, error) {
	ceData := toCloudEventData(&msg.EventData)

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetSource(srcAttr)
	event.SetType(typeAttr)
	if err := event.SetData(cloudevents.ApplicationJSON, ceData); err != nil {
		return nil, fmt.Errorf("setting CloudEvent data: %w", err)
	}

	return &event, nil
}

// toCloudEventData returns a eventhub.Event in a shape that is suitable for
// JSON serialization inside some CloudEvent data.
func toCloudEventData(e *azeventhubs.EventData) interface{} {
	var data interface{}
	data = e

	// if event.Body contains raw JSON data, type it as json.RawMessage so
	// it doesn't get encoded to base64 during the serialization of the
	// CloudEvent data.
	var rawData json.RawMessage
	if err := json.Unmarshal(e.Body, &rawData); err == nil {
		data = &EventWithRawData{
			Body:      rawData,
			EventData: e,
		}
	}

	return data
}

// EventWithRawData is an eventhub.Event with RawMessage-typed data.
type EventWithRawData struct {
	Body json.RawMessage
	*azeventhubs.EventData
}

// eventGridMessageProcessor processes events originating from Azure Event Grid.
type eventGridMessageProcessor struct {
	// These values are set respectively as the "source" and "type" CE
	// context attributes when the message processor handles data which
	// didn't originate from Event Grid.
	ceSourceFallback string
	ceTypeFallback   string
}

// Process implements MessageProcessor.
//
// This processor discards everything from the given message except its data,
// which is expected to be a slice of CloudEvents serialized as JSON.
//
// Expected structure of events in a slice:
// https://docs.microsoft.com/en-us/azure/event-grid/cloud-event-schema
func (p *eventGridMessageProcessor) Process(msg *azeventhubs.ReceivedEventData) ([]*cloudevents.Event, error) {
	events := make([]*cloudevents.Event, 0)

	if err := json.Unmarshal(msg.Body, &events); err != nil {
		// the message didn't originate from Event Grid, fall back to
		// the default processor's behaviour
		event, err := makeEventHubsEvent(msg, p.ceSourceFallback, p.ceTypeFallback)
		if err != nil {
			return nil, fmt.Errorf("creating CloudEvent from Event Hubs message: %w", err)
		}

		return []*cloudevents.Event{event}, nil
	}

	// NOTE(antoineco): Azure Event Grid currently doesn't set the
	// "datacontenttype" context attribute, yet it only sends JSON payloads
	// according to Azure's documentation.
	// Although this attribute is optional, we set it here manually for
	// event consumers.
	for _, event := range events {
		if event.DataContentType() == "" {
			event.SetDataContentType(cloudevents.ApplicationJSON)
		}
	}

	return events, nil
}
