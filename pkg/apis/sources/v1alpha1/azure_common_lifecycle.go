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

package v1alpha1

// AzureEventType returns an event type in a format suitable for usage as a
// CloudEvent type attribute.
func AzureEventType(service, eventType string) string {
	return "com.microsoft.azure." + service + "." + eventType
}

// Reasons for status conditions
const (
	// AzureReasonNoClient is set on a status condition when an Azure API client cannot be obtained.
	AzureReasonNoClient = "NoClient"
	// AzureReasonAPIError is set on a status condition when an Azure API returns an error.
	AzureReasonAPIError = "APIError"
)
