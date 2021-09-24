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

package googlecloudpubsubsource

import (
	"encoding/json"
	"fmt"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"cloud.google.com/go/pubsub"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
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
}

// Process implements MessageProcessor.
func (p *defaultMessageProcessor) Process(msg *pubsub.Message) ([]*cloudevents.Event, error) {
	event, err := makePubSubEvent(msg, p.ceSource)
	if err != nil {
		return nil, fmt.Errorf("creating CloudEvent from Pub/Sub message: %w", err)
	}

	return []*cloudevents.Event{event}, nil
}

// makePubSubEvent returns a CloudEvent for a generic Pub/Sub message.
func makePubSubEvent(msg *pubsub.Message, srcAttr string) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent()
	event.SetID(msg.ID)
	event.SetTime(msg.PublishTime)
	event.SetType(v1alpha1.GoogleCloudPubSubGenericEventType)
	event.SetSource(srcAttr)

	for attr, v := range msg.Attributes {
		attr = sanitizePubSubMessageAttribute(attr)
		event.SetExtension(attr, v)
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
	data = msg.Data

	// if msg.Data contains raw JSON data, type it as json.RawMessage so
	// it doesn't get encoded to base64 during the serialization of the
	// CloudEvent data.
	if json.Valid(msg.Data) {
		data = json.RawMessage(msg.Data)
	}

	return data
}

// sanitizePubSubMessageAttribute returns the given Pub/Sub message attribute
// in a format that is compatible with the CloudEvent spec.
// Ref. https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#extension-context-attributes
func sanitizePubSubMessageAttribute(attr string) string {
	attr = stripNonAlphanumCharsAndMapToLower(attr)

	if isReservedCEAttribute(attr) {
		attr = "x" + attr
	}

	return attr
}

// stripNonAlphanumCharsAndMapToLower applies the following transformations to
// the given string:
//  - strips all non alphanumeric characters
//  - maps all Unicode letters to their lower case
func stripNonAlphanumCharsAndMapToLower(s string) string {
	var stripped strings.Builder
	stripped.Grow(len(s))

	// operate on bytes instead of runes, since all alphanumeric characters
	// are represented in a single byte
	for i := 0; i < len(s); i++ {
		b := s[i]

		if ('a' <= b && b <= 'z') ||
			('A' <= b && b <= 'Z') ||
			('0' <= b && b <= '9') {

			if 'A' <= b && b <= 'Z' {
				// shift from upper to lower case
				b += 'a' - 'A'
			}

			stripped.WriteByte(b)
		}
	}

	return stripped.String()
}

// isReservedCEAttribute returns whether the given Pub/Sub messages attribute
// is a reserved CloudEvent context attribute.
func isReservedCEAttribute(attr string) bool {
	return reservedCEAttributes.has(attr)
}

// reservedCEAttributes is a set of reserved CloudEvent context attributes.
var reservedCEAttributes = initReservedCEAttributes()

// initReservedCEAttributes returns a set containing all reserved CloudEvent
// context attributes, as of CloudEvents v1.0.1.
// Ref. https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#context-attributes.
//
// These attributes often have special constraints (RFC, etc.), so we disallow
// attributes from Pub/Sub messages to overlap with CloudEvent attributes from
// this set.
func initReservedCEAttributes() stringSet {
	reserved := []string{
		"id",
		"source",
		"specversion",
		"type",
		"datacontenttype",
		"dataschema",
		"subject",
		"time",
	}

	rs := make(stringSet, len(reserved))

	for _, attr := range reserved {
		rs.add(attr)
	}

	return rs
}
