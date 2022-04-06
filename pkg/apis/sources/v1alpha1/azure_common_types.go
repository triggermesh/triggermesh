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

// AzureResourceID represents a resource ID for an Azure resource.
type AzureResourceID struct {
	SubscriptionID   string
	ResourceGroup    string
	ResourceProvider string
	Namespace        string
	ResourceType     string
	ResourceName     string
	SubResourceType  string
	SubResourceName  string
}

var (
	_ fmt.Stringer     = (*AzureResourceID)(nil)
	_ json.Marshaler   = (*AzureResourceID)(nil)
	_ json.Unmarshaler = (*AzureResourceID)(nil)
)

const (
	azureResourceIDFormat = "/subscriptions/{subscriptionId}" +
		"[/resourceGroups/{resourceGroupName}" +
		"[/providers/{resourceProviderNamespace}" +
		"[/namespaces/{namespaceName}]" +
		"/{resourceType}/{resourceName}" +
		"[/{subresourceType}/{subresourceName}]]]"

	// Subscription
	//   /subscriptions/s
	azureSubscriptionResourceIDSplitElements = 3
	// Resource group
	//   /subscriptions/s/resourceGroups/rg
	azureResourceGroupResourceIDSplitElements = 5
	// Resource (including namespaces)
	//   /subscriptions/s/resourceGroups/rg/providers/rp/rt/rn
	//   /subscriptions/s/resourceGroups/rg/providers/rp/namespaces/ns
	azureResourceResourceIDSplitElements = 9
	// Resource with subresource (including namespaced resource)
	//   /subscriptions/s/resourceGroups/rg/providers/rp/rt/rn/srt/srn
	//   /subscriptions/s/resourceGroups/rg/providers/rp/namespaces/ns/rt/rn
	azureSubResourceResourceIDSplitElements = 11
	// Namespaced resource with subresource
	//   /subscriptions/s/resourceGroups/rg/providers/rp/namespaces/ns/rt/rn/srt/srn
	azureNamespacedSubResourceResourceIDSplitElements = 13
)

// UnmarshalJSON implements json.Unmarshaler
func (rID *AzureResourceID) UnmarshalJSON(data []byte) error {
	var dataStr string
	if err := json.Unmarshal(data, &dataStr); err != nil {
		return err
	}

	sections := strings.Split(dataStr, "/")
	if n := len(sections); n != azureSubscriptionResourceIDSplitElements &&
		n != azureResourceGroupResourceIDSplitElements &&
		n != azureResourceResourceIDSplitElements &&
		n != azureSubResourceResourceIDSplitElements &&
		n != azureNamespacedSubResourceResourceIDSplitElements {

		return newParseAzureResourceIDError(dataStr)
	}

	const (
		subscriptionIDIdx   = 2
		resourceGroupIdx    = 4
		resourceProviderIdx = 6
		resourceTypeIdx     = 7
		resourceNameIdx     = 8
		subresourceTypeIdx  = 9
		subresourceNameIdx  = 10
		// with namespace
		namespaceIdx         = 8
		resourceTypeNsIdx    = 9
		resourceNameNsIdx    = 10
		subresourceTypeNsIdx = 11
		subresourceNameNsIdx = 12
	)

	// An Azure resource ID always includes a subscription ID. Whether
	// other elements should be defined in the resource ID depends on the
	// type of resource that the ID represents (resource group, resource).
	subscriptionID := sections[subscriptionIDIdx]
	if subscriptionID == "" {
		return errAzureResourceIDEmptyAttrs
	}

	var resourceGroup string
	if len(sections) >= azureResourceGroupResourceIDSplitElements {
		resourceGroup = sections[resourceGroupIdx]
		if resourceGroup == "" {
			return errAzureResourceIDEmptyAttrs
		}
	}

	var resourceProvider string
	var resourceType string
	var resourceName string
	if len(sections) >= azureResourceResourceIDSplitElements {
		resourceProvider = sections[resourceProviderIdx]
		resourceType = sections[resourceTypeIdx]
		resourceName = sections[resourceNameIdx]
		if resourceProvider == "" || resourceType == "" || resourceName == "" {
			return errAzureResourceIDEmptyAttrs
		}
	}

	var namespace string
	var subresourceType string
	var subresourceName string
	if len(sections) >= azureSubResourceResourceIDSplitElements {
		if strings.ToLower(resourceType) == "namespaces" {
			namespace = sections[namespaceIdx]
			resourceType = sections[resourceTypeNsIdx]
			resourceName = sections[resourceNameNsIdx]
			if namespace == "" || resourceType == "" || resourceName == "" {
				return errAzureResourceIDEmptyAttrs
			}
		} else {
			subresourceType = sections[subresourceTypeIdx]
			subresourceName = sections[subresourceNameIdx]
			if subresourceType == "" || subresourceName == "" {
				return errAzureResourceIDEmptyAttrs
			}
		}
	}

	if len(sections) == azureNamespacedSubResourceResourceIDSplitElements {
		subresourceType = sections[subresourceTypeNsIdx]
		subresourceName = sections[subresourceNameNsIdx]
		if subresourceType == "" || subresourceName == "" {
			return errAzureResourceIDEmptyAttrs
		}
	}

	rID.SubscriptionID = subscriptionID
	rID.ResourceGroup = resourceGroup
	rID.ResourceProvider = resourceProvider
	rID.Namespace = namespace
	rID.ResourceType = resourceType
	rID.ResourceName = resourceName
	rID.SubResourceType = subresourceType
	rID.SubResourceName = subresourceName

	return nil
}

// MarshalJSON implements json.Marshaler
func (rID AzureResourceID) MarshalJSON() ([]byte, error) {
	if rID.SubscriptionID == "" {
		return nil, errAzureResourceIDEmptyAttrs
	}

	var b bytes.Buffer

	b.WriteByte('"')
	b.WriteString("/subscriptions/")
	b.WriteString(rID.SubscriptionID)

	if rID.ResourceGroup != "" {
		b.WriteString("/resourceGroups/")
		b.WriteString(rID.ResourceGroup)
	}

	if rID.ResourceProvider != "" || rID.ResourceType != "" || rID.ResourceName != "" {
		// entering this condition means _all_ fields should be set
		if rID.ResourceGroup == "" ||
			rID.ResourceProvider == "" ||
			rID.ResourceType == "" ||
			rID.ResourceName == "" {

			return nil, errAzureResourceIDEmptyAttrs
		}

		b.WriteString("/providers/")
		b.WriteString(rID.ResourceProvider)
		if rID.Namespace != "" {
			b.WriteString("/namespaces/")
			b.WriteString(rID.Namespace)
		}
		b.WriteByte('/')
		b.WriteString(rID.ResourceType)
		b.WriteByte('/')
		b.WriteString(rID.ResourceName)
	}

	if rID.SubResourceType != "" || rID.SubResourceName != "" {
		// entering this condition means _all_ fields should be set
		if rID.SubResourceType == "" ||
			rID.SubResourceName == "" {

			return nil, errAzureResourceIDEmptyAttrs
		}

		b.WriteByte('/')
		b.WriteString(rID.SubResourceType)
		b.WriteByte('/')
		b.WriteString(rID.SubResourceName)
	}

	b.WriteByte('"')

	return b.Bytes(), nil
}

// String implements the fmt.Stringer interface.
func (rID *AzureResourceID) String() string {
	b, err := rID.MarshalJSON()
	if err != nil {
		return ""
	}

	// skip checks on slice bound and leading/trailing quotes since we know
	// exactly what MarshalJSON returns
	return string(b[1 : len(b)-1])
}

// errAzureResourceIDEmptyAttrs indicates that a resource ID
// string or object contains empty attributes.
var errAzureResourceIDEmptyAttrs = errors.New("resource ID contains empty attributes")

// errParseAzureResourceID indicates that a resource ID string
// does not match the expected format.
type errParseAzureResourceID struct {
	gotInput string
}

func newParseAzureResourceIDError(got string) error {
	return &errParseAzureResourceID{
		gotInput: got,
	}
}

// Error implements the error interface.
func (e *errParseAzureResourceID) Error() string {
	return fmt.Sprintf("resource ID %q does not match expected format %q",
		e.gotInput, azureResourceIDFormat)
}
