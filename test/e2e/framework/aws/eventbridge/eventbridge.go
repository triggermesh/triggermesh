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

// Package eventbridge contains helpers for AWS EventBridge.
package eventbridge

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/aws/aws-sdk-go/service/eventbridge/eventbridgeiface"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	e2eaws "github.com/triggermesh/triggermesh/test/e2e/framework/aws"
)

// CreateEventBus creates an event bus.
func CreateEventBus(ebClient eventbridgeiface.EventBridgeAPI, f *framework.Framework) string /*arn*/ {
	eventBus := &eventbridge.CreateEventBusInput{
		Name: &f.UniqueName,
		Tags: tagsAsEventBridgeTags(e2eaws.TagsFor(f)),
	}

	resp, err := ebClient.CreateEventBus(eventBus)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create event bus %q: %s", *eventBus.Name, err)
	}

	return *resp.EventBusArn
}

// DeleteEventBus deletes an event bus.
// To be eligible for deletion, the event bus MUST NOT contain any rule. See also ForceDeleteEventBus.
func DeleteEventBus(ebClient eventbridgeiface.EventBridgeAPI, eventBusName string) {
	deleteEventBus(ebClient, eventBusName, false)
}

// ForceDeleteEventBus deletes an event bus and all its rules.
func ForceDeleteEventBus(ebClient eventbridgeiface.EventBridgeAPI, eventBusName string) {
	deleteEventBus(ebClient, eventBusName, true)
}

// deleteEventBus deletes an event bus. The force parameter controls whether
// any associated rule should also be deleted.
func deleteEventBus(ebClient eventbridgeiface.EventBridgeAPI, eventBusName string, force bool) {
	if force {
		deleteAllRules(ebClient, eventBusName)
	}

	eventBus := &eventbridge.DeleteEventBusInput{
		Name: &eventBusName,
	}

	if _, err := ebClient.DeleteEventBus(eventBus); err != nil {
		framework.FailfWithOffset(2, "Failed to delete event bus %q: %s", eventBusName, err)
	}
}

// SendMessage sends a message to the given event bus.
func SendMessage(ebClient eventbridgeiface.EventBridgeAPI, eventBusName string) string /*event id*/ {
	msg := &eventbridge.PutEventsRequestEntry{
		EventBusName: &eventBusName,
		Detail:       aws.String(`{"msg:": "hello, world!"}`),
		DetailType:   aws.String("e2e.test"),
		Source:       aws.String("e2e.triggermesh"),
	}

	in := &eventbridge.PutEventsInput{
		Entries: []*eventbridge.PutEventsRequestEntry{msg},
	}

	resp, err := ebClient.PutEvents(in)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to send message to event bus %q: %s", *msg.EventBusName, err)
	}

	return *resp.Entries[0].EventId
}
