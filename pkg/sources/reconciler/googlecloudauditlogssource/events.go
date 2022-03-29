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

package googlecloudauditlogssource

const (
	// ReasonSubscribed indicates that a source subscribed to change
	// notifications from a Cloud Audit Logs Sink.
	ReasonSubscribed = "Subscribed"
	// ReasonUnsubscribed indicates that a source unsubscribed from change
	// notifications from a Cloud Audit Logs Sink.
	ReasonUnsubscribed = "Unsubscribed"
	// ReasonFailedSubscribe indicates a failure while synchronizing the
	// notification configuration of a Cloud Audit Logs Sink, or the Pub/Sub
	// subscription it depends on.
	ReasonFailedSubscribe = "FailedSubscribe"
	// ReasonFailedUnsubscribe indicates a failure while deleting the
	// notification configuration of a Cloud Audit Logs Sink, or the Pub/Sub
	// subscription it depends on.
	ReasonFailedUnsubscribe = "FailedUnsubscribe"
)
