package azuresentineltarget

import (
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

type envAccessor struct {
	pkgadapter.EnvConfig
	SubscriptionID string `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	ResourceGroup  string `envconfig:"AZURE_RESOURCE_GROUP" required:"true"`
	Workspace      string `envconfig:"AZURE_WORKSPACE" required:"true"`
	ClientSecret   string `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	ClientID       string `envconfig:"AZURE_CLIENT_ID" required:"true"`
	TenantID       string `envconfig:"AZURE_TENANT_ID" required:"true"`
	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
	// Sink defines the target sink for the events. If no Sink is defined the
	// events are replied back to the sender.
	Sink string `envconfig:"K_SINK"`
}

type IncidentLabel struct {
	LabelName string            `json:"labelName"`
	LabelType IncidentLabelType `json:"labelType"`
}

type IncidentLabelType []struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type IncidentOwnerInfo struct {
	// ObjectId string `json:"objectId"`
	AssignedTo string `json:"assignedTo"`
}

type IncidentStatus struct {
	Active string `json:"active"`
	Closed string `json:"closed"`
	New    string `json:"new"`
}

type Incident struct {
	Etag       string `json:"etag"`
	Properties struct {
		LastActivityTimeUtc  string            `json:"lastActivityTimeUtc"`
		FirstActivityTimeUtc string            `json:"firstActivityTimeUtc"`
		Labels               []IncidentLabel   `json:"labels"`
		Owner                IncidentOwnerInfo `json:"owner"`
		ProviderIncidentId   string            `json:"providerIncidentId"`
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
