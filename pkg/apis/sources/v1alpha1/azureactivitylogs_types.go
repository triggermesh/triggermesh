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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureActivityLogsSource is the Schema for the event source.
type AzureActivityLogsSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureActivityLogsSourceSpec   `json:"spec,omitempty"`
	Status AzureActivityLogsSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AzureActivityLogsSource)(nil)
	_ v1alpha1.AdapterConfigurable = (*AzureActivityLogsSource)(nil)
	_ v1alpha1.EventSource         = (*AzureActivityLogsSource)(nil)
	_ v1alpha1.EventSender         = (*AzureActivityLogsSource)(nil)
)

// AzureActivityLogsSourceSpec defines the desired state of the event source.
type AzureActivityLogsSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// The ID of the Azure subscription which activity logs to subscribe to.
	SubscriptionID string `json:"subscriptionID"`

	// The intermediate destination of activity logs, before they are
	// retrieved by this event source.
	Destination AzureActivityLogsSourceDestination `json:"destination"`

	// Categories of Activity Logs to collect.
	//
	// All available categories are selected when this attribute is empty.
	// https://docs.microsoft.com/en-us/azure/azure-monitor/platform/activity-log-schema#categories
	//
	// +optional
	Categories []string `json:"categories,omitempty"`

	// Authentication method to interact with the Azure Monitor REST API.
	// This event source only supports the ServicePrincipal authentication.
	// If it not present, it will try to use Azure AKS Managed Identity
	Auth AzureAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// AzureActivityLogsSourceDestination contains possible intermediate
// destinations for activity logs.
type AzureActivityLogsSourceDestination struct {
	EventHubs AzureActivityLogsSourceDestinationEventHubs `json:"eventHubs"`
}

// AzureActivityLogsSourceDestinationEventHubs contains properties of an Event
// Hubs namespace to use as intermediate destination for events.
type AzureActivityLogsSourceDestinationEventHubs struct {
	// Resource ID of the Event Hubs namespace.
	//
	// The expected format is
	//   /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}
	NamespaceID AzureResourceID `json:"namespaceID"`

	// Name of the Event Hubs instance within the selected namespace. If
	// omitted, Azure automatically creates an Event Hub with the name
	// 'insights-activity-logs' inside the selected namespace.
	//
	// +optional
	HubName *string `json:"hubName,omitempty"`

	// Name of the Event Hubs' Consumer Group that will be used by the source to read the event stream.
	ConsumerGroup *string `json:"consumerGroup,omitempty"`

	// Name of a SAS policy with Manage permissions inside the Event Hubs
	// namespace referenced in the EventHubID field.
	//
	// Defaults to "RootManageSharedAccessKey".
	//
	// References:
	//  * https://docs.microsoft.com/en-us/rest/api/eventhub/2017-04-01/authorization%20rules%20-%20namespaces/getauthorizationrule
	//  * https://docs.microsoft.com/en-us/azure/event-hubs/authorize-access-shared-access-signature
	//
	// +optional
	SASPolicy *string `json:"sasPolicy,omitempty"`
}

// AzureActivityLogsSourceStatus defines the observed state of the event source.
type AzureActivityLogsSourceStatus struct {
	v1alpha1.Status `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureActivityLogsSourceList contains a list of event sources.
type AzureActivityLogsSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureActivityLogsSource `json:"items"`
}
