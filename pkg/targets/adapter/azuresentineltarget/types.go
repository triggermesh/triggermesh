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

package azuresentineltarget

import (
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

type envAccessor struct {
	pkgadapter.EnvConfig
	// SubscriptionID refers to the Azure Subscription ID that the Azure Sentinel instance is associated with.
	SubscriptionID string `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	// ResourceGroup refers to the resource group where the Azure Sentinel instance is deployed.
	ResourceGroup string `envconfig:"AZURE_RESOURCE_GROUP" required:"true"`
	// Workspace refers to the workspace name in Azure Sentinel.
	Workspace string `envconfig:"AZURE_WORKSPACE" required:"true"`
	// ClientSecret refers to the Client Secret of an App Registration. see -> https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app for more details
	ClientSecret string `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	// ClientID refers to the Application (client) ID of the App Registration. see -> https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app for more details
	ClientID string `envconfig:"AZURE_CLIENT_ID" required:"true"`
	// TenantID refers to the Directory (tenant) ID of the App Registration. see -> https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app for more details
	TenantID string `envconfig:"AZURE_TENANT_ID" required:"true"`
	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
}

// IncidentLabel is the label used to identify the incident in the Azure Sentinel
type IncidentLabel struct {
	LabelName string            `json:"labelName"`
	LabelType IncidentLabelType `json:"labelType"`
}

// IncidentLabelType is the type of the label associated with an incident
type IncidentLabelType []struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// IncidentOwnerInfo is the owner information of an incident
type IncidentOwnerInfo struct {
	// ObjectId string `json:"objectId"`
	AssignedTo string `json:"assignedTo"`
}

// IncidentStatus an Azure Sentinel incident status object
type IncidentStatus struct {
	Active string `json:"active"`
	Closed string `json:"closed"`
	New    string `json:"new"`
}

// Incident an Azure Sentinel incident.
type Incident struct {
	Etag       string `json:"etag"`
	Properties struct {
		LastActivityTimeUtc  string            `json:"lastActivityTimeUtc"`
		FirstActivityTimeUtc string            `json:"firstActivityTimeUtc"`
		Labels               []IncidentLabel   `json:"labels"`
		Owner                IncidentOwnerInfo `json:"owner"`
		ProviderIncidentID   string            `json:"providerIncidentId"`
		Severity             string            `json:"severity"`
		Status               string            `json:"status"`
		Title                string            `json:"title"`
		Description          string            `json:"description"`
		AdditionalData       struct {
			AlertProductNames []string `json:"alertProductNames"`
		} `json:"additionalData"`
		// Labels IncidentLabel `json:"labels"`
	} `json:"properties"`
}
