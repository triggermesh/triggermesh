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

const (
	// ReasonSubscribed indicates that a source subscribed to a Pub/Sub topic.
	ReasonSubscribed = "Subscribed"
	// ReasonUnsubscribed indicates that a source unsubscribed from a Pub/Sub topic.
	ReasonUnsubscribed = "Unsubscribed"
	// ReasonFailedSubscribe indicates a failure while synchronizing a Pub/Sub subscription.
	ReasonFailedSubscribe = "FailedSubscribe"
	// ReasonFailedUnsubscribe indicates a failure while deleting a Pub/Sub subscription.
	ReasonFailedUnsubscribe = "FailedUnsubscribe"
)
