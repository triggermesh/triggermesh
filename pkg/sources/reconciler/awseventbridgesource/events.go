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

package awseventbridgesource

const (
	// ReasonQueueCreated indicates that a SQS queue was created for receiving EventBridge events.
	ReasonQueueCreated = "QueueCreated"
	// ReasonQueueDeleted indicates that a SQS queue used for receiving EventBridge events was deleted.
	ReasonQueueDeleted = "QueueDeleted"
	// ReasonFailedQueue indicates a failure while synchronizing the SQS queue for receiving EventBridge events.
	ReasonFailedQueue = "FailedQueue"

	// ReasonSubscribed indicates that events from an EventBridge event bus have been successfully subscribed to via
	// a rule.
	ReasonSubscribed = "Subscribed"
	// ReasonUnsubscribed indicates that the subscription to an EventBridge event bus has been terminated by
	// removing a rule.
	ReasonUnsubscribed = "Unsubscribed"
	// ReasonFailedSubscribe indicates a failure while subscribing to events from an EventBridge event bus.
	ReasonFailedSubscribe = "FailedSubscribe"
	// ReasonFailedUnsubscribe indicates a failure while unsubscribing from events from an EventBridge event bus.
	ReasonFailedUnsubscribe = "FailedUnsubscribe"
)
