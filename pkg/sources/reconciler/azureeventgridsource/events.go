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

package azureeventgridsource

const (
	// ReasonSystemTopicSynced indicates that a system topic was created or updated.
	ReasonSystemTopicSynced = "SystemTopicSynced"
	// ReasonSystemTopicFinalized indicates that a system topic was finalized.
	ReasonSystemTopicFinalized = "SystemTopicFinalized"
	// ReasonFailedSystemTopic indicates a failure while synchronizing a system topic.
	ReasonFailedSystemTopic = "FailedSystemTopic"

	// ReasonSubscribed indicates that an event subscription was created or updated inside a system topic.
	ReasonSubscribed = "Subscribed"
	// ReasonUnsubscribed indicates that an event subscription was removed from a system topic.
	ReasonUnsubscribed = "Unsubscribed"
	// ReasonFailedSubscribe indicates a failure while synchronizing an event subscription in a system topic.
	ReasonFailedSubscribe = "FailedSubscribe"
	// ReasonFailedUnsubscribe indicates a failure while removing an event subscription from a system topic.
	ReasonFailedUnsubscribe = "FailedUnsubscribe"

	// ReasonEventHubCreated indicates that an Event Hub was created for receiving events.
	ReasonEventHubCreated = "EventHubCreated"
	// ReasonEventHubDeleted indicates that an Event Hub used for receiving events was deleted.
	ReasonEventHubDeleted = "EventHubDeleted"
	// ReasonFailedEventHub indicates a failure while synchronizing the Event Hub for receiving events.
	ReasonFailedEventHub = "FailedEventHub"

	// ReasonResourceGroupCreated indicates that a resource group was created.
	ReasonResourceGroupCreated = "ResourceGroupCreated"
	// ReasonFailedEventHub indicates a failure while synchronizing a resource group.
	ReasonFailedResourceGroup = "FailedResourceGroup"
)
