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

package awss3source

const (
	// ReasonQueueCreated indicates that a SQS queue was created for receiving S3 event notifications.
	ReasonQueueCreated = "QueueCreated"
	// ReasonQueueDeleted indicates that a SQS queue used for receiving S3 events was deleted.
	ReasonQueueDeleted = "QueueDeleted"
	// ReasonFailedQueue indicates a failure while synchronizing the SQS queue for receiving S3 event notifications.
	ReasonFailedQueue = "FailedQueue"

	// ReasonSubscribed indicates that event notifications were enabled for a S3 bucket.
	ReasonSubscribed = "Subscribed"
	// ReasonUnsubscribed indicates that event notifications were disabled for a S3 bucket.
	ReasonUnsubscribed = "Unsubscribed"
	// ReasonFailedSubscribe indicates a failure while enabling event notifications for a S3 bucket.
	ReasonFailedSubscribe = "FailedSubscribe"
	// ReasonFailedUnsubscribe indicates a failure while disabling event notifications for a S3 bucket.
	ReasonFailedUnsubscribe = "FailedUnsubscribe"
)
