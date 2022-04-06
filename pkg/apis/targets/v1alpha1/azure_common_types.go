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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// AzureAuth contains multiple authentication methods for Azure services.
type AzureAuth struct {
	// Service principals provide a way to create a non-interactive account
	// associated with your identity to which you grant only the privileges
	// your app needs to run.
	// See https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals
	ServicePrincipal *AzureServicePrincipal `json:"servicePrincipal,omitempty"`

	// A shared access signature (SAS) provides secure delegated access to
	// resources in a storage account.
	// See https://docs.microsoft.com/en-us/azure/storage/common/storage-sas-overview
	SASToken *AzureSASToken `json:"sasToken,omitempty"`
}

// AzureServicePrincipal represents an AAD Service Principal.
type AzureServicePrincipal struct {
	TenantID     v1alpha1.ValueFromField `json:"tenantID"`
	ClientID     v1alpha1.ValueFromField `json:"clientID"`
	ClientSecret v1alpha1.ValueFromField `json:"clientSecret"`
}

// AzureSASToken represents an Azure SAS token.
type AzureSASToken struct {
	KeyName          v1alpha1.ValueFromField `json:"keyName"`
	KeyValue         v1alpha1.ValueFromField `json:"keyValue"`
	ConnectionString v1alpha1.ValueFromField `json:"connectionString"`
}

// EventHubResourceID represents a resource ID for an Event Hubs instance or namespace.
type EventHubResourceID struct {
	SubscriptionID string
	ResourceGroup  string
	Namespace      string
	EventHub       string
}

var (
	_ fmt.Stringer     = (*EventHubResourceID)(nil)
	_ json.Marshaler   = (*EventHubResourceID)(nil)
	_ json.Unmarshaler = (*EventHubResourceID)(nil)
)

const (
	eventHubResourceIDFormat = "/subscriptions/{subscriptionId}" +
		"/resourceGroups/{resourceGroupName}" +
		"/providers/Microsoft.EventHub" +
		"/namespaces/{namespaceName}" +
		"[/eventHubs/{eventHubName}]"

	eventHubResourceIDSplitElements    = 11
	eventHubsNsResourceIDSplitElements = 9
)

// UnmarshalJSON implements json.Unmarshaler
func (rID *EventHubResourceID) UnmarshalJSON(data []byte) error {
	var dataStr string
	if err := json.Unmarshal(data, &dataStr); err != nil {
		return err
	}

	sections := strings.Split(dataStr, "/")
	if n := len(sections); n != eventHubResourceIDSplitElements && n != eventHubsNsResourceIDSplitElements {
		return newParseEventHubResourceIDError(dataStr)
	}

	const (
		subscriptionIDIdx = 2
		resourceGroupIdx  = 4
		namespaceIdx      = 8
		eventHubIdx       = 10
	)

	subscriptionID := sections[subscriptionIDIdx]
	resourceGroup := sections[resourceGroupIdx]
	namespace := sections[namespaceIdx]

	// the eventHub element can be empty, in which case the resource ID
	// represents an Event Hubs namespace
	var eventHub string
	if len(sections) == eventHubResourceIDSplitElements {
		eventHub = sections[eventHubIdx]
	}

	if subscriptionID == "" || resourceGroup == "" || namespace == "" {
		return errEventHubResourceIDEmptyAttrs
	}

	rID.SubscriptionID = subscriptionID
	rID.ResourceGroup = resourceGroup
	rID.Namespace = namespace
	rID.EventHub = eventHub

	return nil
}

// MarshalJSON implements json.Marshaler
func (rID EventHubResourceID) MarshalJSON() ([]byte, error) {
	if rID.SubscriptionID == "" || rID.ResourceGroup == "" || rID.Namespace == "" {
		return nil, errEventHubResourceIDEmptyAttrs
	}

	var b bytes.Buffer

	b.WriteByte('"')
	b.WriteString("/subscriptions/")
	b.WriteString(rID.SubscriptionID)
	b.WriteString("/resourceGroups/")
	b.WriteString(rID.ResourceGroup)
	b.WriteString("/providers/Microsoft.EventHub/namespaces/")
	b.WriteString(rID.Namespace)

	if rID.EventHub != "" {
		b.WriteString("/eventHubs/")
		b.WriteString(rID.EventHub)
	}

	b.WriteByte('"')

	return b.Bytes(), nil
}

// String implements the fmt.Stringer interface.
func (rID *EventHubResourceID) String() string {
	b, err := rID.MarshalJSON()
	if err != nil {
		return ""
	}

	// skip checks on slice bound and leading/trailing quotes since we know
	// exactly what MarshalJSON returns
	return string(b[1 : len(b)-1])
}

// errEventHubResourceIDEmptyAttrs indicates that a resource ID
// string or object contains empty attributes.
var errEventHubResourceIDEmptyAttrs = errors.New("resource ID contains empty attributes")

// errParseEventHubResourceID indicates that a resource ID string
// does not match the expected format.
type errParseEventHubResourceID struct {
	gotInput string
}

func newParseEventHubResourceIDError(got string) error {
	return &errParseEventHubResourceID{
		gotInput: got,
	}
}

// Error implements the error interface.
func (e *errParseEventHubResourceID) Error() string {
	return fmt.Sprintf("Event Hub resource ID %q does not match expected format %q",
		e.gotInput, eventHubResourceIDFormat)
}
