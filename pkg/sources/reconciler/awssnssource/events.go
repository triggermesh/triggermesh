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

package awssnssource

const (
	// ReasonSubscribed indicates the successful creation of a SNS subscription.
	ReasonSubscribed = "Subscribed"
	// ReasonUnsubscribed indicates the successful deletion of a SNS subscription.
	ReasonUnsubscribed = "Unsubscribed"
	// ReasonFailedSubscribe indicates a failure during the subscription to a SNS topic.
	ReasonFailedSubscribe = "FailedSubscribe"
	// ReasonFailedUnsubscribe indicates a failure during the deletion of a SNS subscription.
	ReasonFailedUnsubscribe = "FailedUnsubscribe"
)
