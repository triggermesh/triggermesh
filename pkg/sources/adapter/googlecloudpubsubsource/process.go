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

package googlecloudpubsubsource

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"cloud.google.com/go/pubsub"
)

// MessageProcessor converts a Pub/Sub message to a CloudEvent.
type MessageProcessor interface {
	Process(*pubsub.Message) ([]*cloudevents.Event, error)
}

var (
	_ MessageProcessor = (*defaultMessageProcessor)(nil)
)

// defaultMessageProcessor is the default processor for Pub/Sub messages.
type defaultMessageProcessor struct {
	ceSource string
	ceType   string
}

// Process implements MessageProcessor.
func (p *defaultMessageProcessor) Process(msg *pubsub.Message) ([]*cloudevents.Event, error) {
	event, err := makePubSubEvent(msg, p.ceSource, p.ceType)
	if err != nil {
		return nil, fmt.Errorf("creating CloudEvent from Pub/Sub message: %w", err)
	}

	return []*cloudevents.Event{event}, nil
}

// makePubSubEvent returns a CloudEvent for a generic Pub/Sub message.
func makePubSubEvent(msg *pubsub.Message, srcAttr, typeAttr string) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent()
	event.SetID(msg.ID)
	event.SetTime(msg.PublishTime)
	event.SetSource(srcAttr)
	event.SetType(typeAttr)

	for name, val := range ceExtensionAttrsForMessage(msg) {
		event.SetExtension(name, val)
	}

	if err := event.SetData(cloudevents.ApplicationJSON, toCloudEventData(msg)); err != nil {
		return nil, fmt.Errorf("setting CloudEvent data: %w", err)
	}

	return &event, nil
}

// toCloudEventData returns a Pub/Sub message in a shape that is suitable for
// JSON serialization inside some CloudEvent data.
func toCloudEventData(msg *pubsub.Message) interface{} {
	var data interface{}
	data = msg

	// if msg.Data contains raw JSON data, type it as json.RawMessage so
	// it doesn't get encoded to base64 during the serialization of the
	// CloudEvent data.
	if json.Valid(msg.Data) {
		data = &MessageWithRawJSONData{
			Data:    json.RawMessage(msg.Data),
			Message: msg,
		}
	}

	return data
}

// MessageWithRawJSONData is an Message with RawMessage-typed JSON data.
type MessageWithRawJSONData struct {
	Data json.RawMessage
	*pubsub.Message
}
