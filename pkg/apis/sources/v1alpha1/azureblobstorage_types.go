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

package v1alpha1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureBlobStorageSource is the Schema for the event source.
type AzureBlobStorageSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzureBlobStorageSourceSpec   `json:"spec,omitempty"`
	Status AzureBlobStorageSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ runtime.Object = (*AzureBlobStorageSource)(nil)
	_ EventSource    = (*AzureBlobStorageSource)(nil)
)

// AzureBlobStorageSourceSpec defines the desired state of the event source.
type AzureBlobStorageSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Resource ID of the Storage Account to receive events for.
	//
	// Format: /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{storageAccountName}
	//
	// Besides the Storage Account name itself, the resource ID contains
	// the subscription ID and resource group name which all together
	// uniquely identify the Storage Account within Azure.
	StorageAccountID StorageAccountResourceID `json:"storageAccountID"`

	// Types of events to subscribe to.
	//
	// The list of available event types can be found at
	// https://docs.microsoft.com/en-us/azure/event-grid/event-schema-blob-storage
	//
	// When this attribute is not set, the source automatically subscribes
	// to the following event types:
	// - Microsoft.Storage.BlobCreated
	// - Microsoft.Storage.BlobDeleted
	//
	// +optional
	EventTypes []string `json:"eventTypes,omitempty"`

	// The intermediate destination of events subscribed via Event Grid,
	// before they are retrieved by TriggerMesh.
	Endpoint AzureEventGridSourceEndpoint `json:"endpoint"`

	// Authentication method to interact with the Azure REST API.
	// This event source only supports the ServicePrincipal authentication.
	Auth AzureAuth `json:"auth"`
}

// AzureBlobStorageSourceStatus defines the observed state of the event source.
type AzureBlobStorageSourceStatus struct {
	EventSourceStatus `json:",inline"`

	// Resource ID of the Event Hubs instance that is currently receiving
	// events from the Azure Event Grid subscription.
	EventHubID *AzureResourceID `json:"eventHubID,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureBlobStorageSourceList contains a list of event sources.
type AzureBlobStorageSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzureBlobStorageSource `json:"items"`
}

// StorageAccountResourceID represents a resource ID for a Storage Account.
type StorageAccountResourceID struct {
	SubscriptionID string
	ResourceGroup  string
	StorageAccount string
}

var (
	_ fmt.Stringer     = (*StorageAccountResourceID)(nil)
	_ json.Marshaler   = (*StorageAccountResourceID)(nil)
	_ json.Unmarshaler = (*StorageAccountResourceID)(nil)
)

const (
	storageAccountResourceIDFormat = "/subscriptions/{subscriptionId}" +
		"/resourceGroups/{resourceGroupName}" +
		"/providers/Microsoft.Storage" +
		"/storageAccounts/{storageAccountName}"

	storageAccountResourceIDSplitElements = 9
)

// UnmarshalJSON implements json.Unmarshaler
func (rID *StorageAccountResourceID) UnmarshalJSON(data []byte) error {
	var dataStr string
	if err := json.Unmarshal(data, &dataStr); err != nil {
		return err
	}

	sections := strings.Split(dataStr, "/")
	if len(sections) != storageAccountResourceIDSplitElements {
		return newParseStorageAccountResourceIDError(dataStr)
	}

	const (
		subscriptionIDIdx = 2
		resourceGroupIdx  = 4
		storageAccountIdx = 8
	)

	subscriptionID := sections[subscriptionIDIdx]
	resourceGroup := sections[resourceGroupIdx]
	storageAccount := sections[storageAccountIdx]

	if subscriptionID == "" || resourceGroup == "" || storageAccount == "" {
		return errStorageAccountResourceIDEmptyAttrs
	}

	rID.SubscriptionID = subscriptionID
	rID.ResourceGroup = resourceGroup
	rID.StorageAccount = storageAccount

	return nil
}

// MarshalJSON implements json.Marshaler
func (rID StorageAccountResourceID) MarshalJSON() ([]byte, error) {
	if rID.SubscriptionID == "" || rID.ResourceGroup == "" || rID.StorageAccount == "" {
		return nil, errStorageAccountResourceIDEmptyAttrs
	}

	var b bytes.Buffer

	b.WriteByte('"')
	b.WriteString("/subscriptions/")
	b.WriteString(rID.SubscriptionID)
	b.WriteString("/resourceGroups/")
	b.WriteString(rID.ResourceGroup)
	b.WriteString("/providers/Microsoft.Storage/storageAccounts/")
	b.WriteString(rID.StorageAccount)
	b.WriteByte('"')

	return b.Bytes(), nil
}

// String implements the fmt.Stringer interface.
func (rID *StorageAccountResourceID) String() string {
	b, err := rID.MarshalJSON()
	if err != nil {
		return ""
	}

	// skip checks on slice bound and leading/trailing quotes since we know
	// exactly what MarshalJSON returns
	return string(b[1 : len(b)-1])
}

// errStorageAccountResourceIDEmptyAttrs indicates that a resource ID
// string or object contains empty attributes.
var errStorageAccountResourceIDEmptyAttrs = errors.New("resource ID contains empty attributes")

// errParseStorageAccountResourceID indicates that a resource ID string
// does not match the expected format.
type errParseStorageAccountResourceID struct {
	gotInput string
}

func newParseStorageAccountResourceIDError(got string) error {
	return &errParseStorageAccountResourceID{
		gotInput: got,
	}
}

// Error implements the error interface.
func (e *errParseStorageAccountResourceID) Error() string {
	return fmt.Sprintf("Storage Account resource ID %q does not match expected format %q",
		e.gotInput, storageAccountResourceIDFormat)
}
