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

package azureeventgridsource

const (
	// ReasonEventHubCreated indicates that an Event Hub was created for receiving events.
	ReasonEventHubCreated = "EventHubCreated"
	// ReasonEventHubDeleted indicates that an Event Hub used for receiving events was deleted.
	ReasonEventHubDeleted = "EventHubDeleted"
	// ReasonFailedEventHub indicates a failure while synchronizing the Event Hub for receiving events.
	ReasonFailedEventHub = "FailedEventHub"

	// ReasonSubscribed indicates that an event subscription was enabled for an Azure resource.
	ReasonSubscribed = "Subscribed"
	// ReasonUnsubscribed indicates that an event subscription was removed for an Azure resource.
	ReasonUnsubscribed = "Unsubscribed"
	// ReasonFailedSubscribe indicates a failure while synchronizing an event subscription for an Azure resource.
	ReasonFailedSubscribe = "FailedSubscribe"
	// ReasonFailedUnsubscribe indicates a failure while removing an event subscription for an Azure resource.
	ReasonFailedUnsubscribe = "FailedUnsubscribe"
)
