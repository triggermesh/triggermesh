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

package awssqssource

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

// A message processor processes SQS messages (sends as CloudEvent)
// sequentially, as soon as they are written to processQueue.
func (a *adapter) runMessagesProcessor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-a.processQueue:
			a.sr.reportMessageDequeuedProcessCount()

			a.logger.Debugw("Processing message", zap.String(logfieldMsgID, *msg.MessageId))

			events, err := a.msgPrcsr.Process(msg)
			if err != nil {
				a.logger.Errorw("Failed to process SQS message", zap.Error(err),
					zap.String(logfieldMsgID, *msg.MessageId))
				continue
			}

			sendError := false
			for _, event := range events {
				if err := sendSQSEvent(ctx, a.ceClient, event); err != nil {
					a.logger.Errorw("Failed to send event to the sink", zap.Error(err))
					sendError = true
					break
				}
			}

			if sendError {
				continue
			}

			a.deleteQueue <- msg
			a.sr.reportMessageEnqueuedDeleteCount()
		}
	}
}

// sendSQSEvent sends a single SQS message as a CloudEvent to the event sink.
func sendSQSEvent(ctx context.Context, cli cloudevents.Client, event *cloudevents.Event) error {
	if result := cli.Send(ctx, *event); !cloudevents.IsACK(result) {
		return result
	}
	return nil
}

// MessageProcessor converts SQS messages to CloudEvents.
type MessageProcessor interface {
	Process(*sqs.Message) ([]*cloudevents.Event, error)
}

var (
	_ MessageProcessor = (*defaultMessageProcessor)(nil)
	_ MessageProcessor = (*s3MessageProcessor)(nil)
)

// defaultMessageProcessor is the default message processor.
type defaultMessageProcessor struct {
	ceSource string
}

// Process implements MessageProcessor.
func (p *defaultMessageProcessor) Process(msg *sqs.Message) ([]*cloudevents.Event, error) {
	event, err := makeSQSEvent(msg, p.ceSource)
	if err != nil {
		return nil, fmt.Errorf("creating CloudEvent from SQS message: %w", err)
	}

	return []*cloudevents.Event{event}, nil
}

// makeSQSEvent returns a CloudEvent for a generic SQS message.
func makeSQSEvent(msg *sqs.Message, srcAttr string) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent()
	event.SetType(v1alpha1.AWSEventType(sqs.ServiceName, v1alpha1.AWSSQSGenericEventType))
	event.SetSource(srcAttr)
	event.SetID(*msg.MessageId)

	for name, val := range ceExtensionAttrsForMessage(msg) {
		event.SetExtension(name, val)
	}

	if err := event.SetData(cloudevents.ApplicationJSON, toCloudEventData(msg)); err != nil {
		return nil, fmt.Errorf("setting CloudEvent data: %w", err)
	}

	return &event, nil
}

// toCloudEventData returns a SQS message in a shape that is suitable for JSON
// serialization inside some CloudEvent data.
func toCloudEventData(msg *sqs.Message) interface{} {
	if msg.Body == nil {
		return msg
	}

	var data interface{}
	data = msg

	// if msg.Body contains raw JSON data, type it as json.RawMessage so it
	// doesn't get encoded to base64 during the serialization of the
	// CloudEvent data.
	if json.Valid([]byte(*msg.Body)) {
		data = &messageWithRawJSONBody{
			Body:    json.RawMessage([]byte(*msg.Body)),
			Message: msg,
		}
	}

	return data
}

// messageWithRawJSONBody is a SQS Message with a RawMessage-typed JSON body.
type messageWithRawJSONBody struct {
	Body json.RawMessage
	*sqs.Message
}

// s3MessageProcessor processes messages originating from S3 buckets.
type s3MessageProcessor struct {
	// this value is set as the "source" CE context attribute when the S3
	// processor handles messages which are not originating from S3
	ceSourceFallback string
}

// Process implements MessageProcessor.
//
// This processor discards everything from the given message except its body,
// which must be in JSON format. If the body contains multiple records, each
// record is converted to an individual event.
//
// Expected events structure: https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-content-structure.html
func (p *s3MessageProcessor) Process(msg *sqs.Message) ([]*cloudevents.Event, error) {
	var events []*cloudevents.Event

	bodyData := make(map[string]interface{})

	if err := json.Unmarshal([]byte(*msg.Body), &bodyData); err != nil {
		// if the data is not a JSON object, we can be certain the
		// message didn't originate from S3, and fall back to the
		// default processor's behaviour
		event, err := makeSQSEvent(msg, p.ceSourceFallback)
		if err != nil {
			return nil, fmt.Errorf("creating CloudEvent from SQS message: %w", err)
		}

		return append(events, event), nil
	}

	var records []interface{}

	recordsVal, hasRecords := bodyData["Records"]
	if hasRecords {
		records, hasRecords = recordsVal.([]interface{})
	}

	switch {
	case hasRecords:
		for _, record := range records {
			event, err := makeS3EventFromRecord(record)
			if err != nil {
				return nil, fmt.Errorf("creating CloudEvent from S3 event record: %w", err)
			}

			events = append(events, event)
		}

	// special case: test events are sent whenever event notifications are
	// re-configured in a S3 bucket
	case isTestEventPayload(bodyData):
		body := json.RawMessage([]byte(*msg.Body))

		event := cloudevents.NewEvent()
		event.SetType(v1alpha1.AWSEventType(s3.ServiceName, v1alpha1.AWSS3TestEventType))
		event.SetSource("arn:aws:s3:::" + bodyData["Bucket"].(string))
		event.SetID(*msg.MessageId)
		if err := event.SetData(cloudevents.ApplicationJSON, body); err != nil {
			return nil, fmt.Errorf("setting CloudEvent data: %w", err)
		}

		events = append(events, &event)

	// instead of discarding non-S3 events, fall back to the default processor's behaviour
	default:
		event, err := makeSQSEvent(msg, p.ceSourceFallback)
		if err != nil {
			return nil, fmt.Errorf("creating CloudEvent from SQS message: %w", err)
		}

		events = append(events, event)

	}

	return events, nil
}

// makeS3EventFromRecord returns a CloudEvent for the given S3 event record.
func makeS3EventFromRecord(record interface{}) (*cloudevents.Event, error) {
	recBytes, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("serializing S3 event record: %w", err)
	}

	data := json.RawMessage(recBytes)

	recordData := record.(map[string]interface{})
	eventName := recordData["eventName"].(string)

	s3prop := recordData["s3"].(map[string]interface{})
	bucketARN := s3prop["bucket"].(map[string]interface{})["arn"].(string)
	objectKey := s3prop["object"].(map[string]interface{})["key"].(string)

	event := cloudevents.NewEvent()
	event.SetType(v1alpha1.AWSEventType(s3.ServiceName, ceTypeFromS3Event(eventName)))
	event.SetSource(bucketARN)
	event.SetSubject(objectKey)
	if err := event.SetData(cloudevents.ApplicationJSON, data); err != nil {
		return nil, fmt.Errorf("setting CloudEvent data: %w", err)
	}

	return &event, nil
}

// ceTypeFromS3Event returns the name of a S3 event in a format that is
// suitable for the "type" context attribute of a CloudEvent.
func ceTypeFromS3Event(eventName string) string {
	// Example: "ObjectRemoved:DeleteMarkerCreated" -> "objectremoved"
	return strings.ToLower(strings.SplitN(eventName, ":", 2)[0])
}

// isTestEventPayload checks whether the provided payload data corresponds to a
// test event from S3.
func isTestEventPayload(data map[string]interface{}) bool {
	v, ok := data["Service"]
	if !ok {
		return false
	}
	if v, ok = v.(string); !ok || v != "Amazon S3" {
		return false
	}

	v, ok = data["Event"]
	if !ok {
		return false
	}
	if v, ok = v.(string); !ok || v != "s3:TestEvent" {
		return false
	}

	return true
}

// eventbridgeMessageProcessor processes messages originating from EventBridge.
type eventbridgeMessageProcessor struct {
	// this value is set as the "source" CE context attribute on messages
	// that originate from EventBridge
	ceSource string
	// this value is set as the "source" CE context attribute when the
	// EventBridge processor handles messages which are not originating
	// from EventBridge
	ceSourceFallback string
}

// Process implements MessageProcessor.
//
// This processor discards everything from the given message except its body,
// which must be in JSON format.
//
// Expected events structure: https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-events.html
func (p *eventbridgeMessageProcessor) Process(msg *sqs.Message) ([]*cloudevents.Event, error) {
	var events []*cloudevents.Event

	bodyData := make(map[string]interface{})

	if err := json.Unmarshal([]byte(*msg.Body), &bodyData); err != nil {
		// if the data is not a JSON object, we can be certain the
		// message didn't originate from EventBridge, and fall back to
		// the default processor's behaviour
		event, err := makeSQSEvent(msg, p.ceSourceFallback)
		if err != nil {
			return nil, fmt.Errorf("creating CloudEvent from SQS message: %w", err)
		}

		return append(events, event), nil
	}

	switch {
	case isEventBridgeEvent(bodyData):
		event, err := makeEventBridgeEvent(bodyData, p.ceSource)
		if err != nil {
			return nil, fmt.Errorf("creating CloudEvent from EventBridge event: %w", err)
		}

		events = append(events, event)

	// instead of discarding non-EventBridge events, fall back to the
	// default processor's behaviour
	default:
		event, err := makeSQSEvent(msg, p.ceSourceFallback)
		if err != nil {
			return nil, fmt.Errorf("creating CloudEvent from SQS message: %w", err)
		}

		events = append(events, event)

	}

	return events, nil
}

// isEventBridgeEvent returns whether the given data represents a valid
// EventBridge event.
func isEventBridgeEvent(data map[string]interface{}) bool {
	if _, ok := data["detail"]; !ok {
		return false
	}
	if _, ok := data["detail-type"]; !ok {
		return false
	}
	_, ok := data["source"]
	return ok
}

// makeEventBridgeEvent returns a CloudEvent for the given EventBridge event.
func makeEventBridgeEvent(data map[string]interface{}, srcAttr string) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent()
	event.SetType(v1alpha1.AWSEventType(eventbridge.EndpointsID, v1alpha1.AWSEventBridgeGenericEventType))
	event.SetSource(srcAttr)
	event.SetExtension("awseventssource", data["source"])
	event.SetExtension("awseventsdetailtype", data["detail-type"])

	if id, ok := data["id"]; ok {
		event.SetID(id.(string))
	}
	if t, ok := data["time"]; ok {
		if ts, err := time.Parse(time.RFC3339, t.(string)); err == nil {
			event.SetTime(ts)
		}
	}

	if err := event.SetData(cloudevents.ApplicationJSON, data); err != nil {
		return nil, fmt.Errorf("setting CloudEvent data: %w", err)
	}

	return &event, nil
}
