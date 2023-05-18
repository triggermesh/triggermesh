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
	"fmt"

	"cloud.google.com/go/pubsub"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

var (
	_ MessageProcessor = (*gcsMessageProcessor)(nil)
)

// gcsMessageProcessor is the GCS events processor for Pub/Sub messages.
type gcsMessageProcessor struct {
	ceSource string
}

// Process implements MessageProcessor.
func (p *gcsMessageProcessor) Process(msg *pubsub.Message) ([]*cloudevents.Event, error) {
	eventType, ok := msg.Attributes["eventType"]
	if !ok {
		eventType = ""
	}
	event, err := p.makePubSubEvent(msg, p.ceSource, v1alpha1.PubSubAttributeToCEType(eventType))
	if err != nil {
		return nil, fmt.Errorf("creating CloudEvent from Pub/Sub message: %w", err)
	}

	return []*cloudevents.Event{event}, nil
}

// makePubSubEvent returns a CloudEvent for a generic Pub/Sub message.
func (p *gcsMessageProcessor) makePubSubEvent(msg *pubsub.Message, srcAttr, typeAttr string) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent()
	event.SetID(msg.ID)
	event.SetTime(msg.PublishTime)
	event.SetSource(srcAttr)
	event.SetType(typeAttr)

	if err := event.SetData(cloudevents.ApplicationJSON, toCloudEventData(msg)); err != nil {
		return nil, fmt.Errorf("setting CloudEvent data: %w", err)
	}

	return &event, nil
}
