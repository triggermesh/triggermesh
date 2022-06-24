package azuresentineltarget

import (
	"time"

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

type Incident struct {
	Properties struct {
		Owner              IncidentOwnerInfo `json:"owner"`
		ProviderIncidentId string            `json:"providerIncidentId"`
		Severity           string            `json:"severity"`
		Status             string            `json:"status"`
		Title              string            `json:"title"`
		Description        string            `json:"description"`
		AdditionalData     struct {
			AlertProductNames []string `json:"alertProductNames"`
		} `json:"additionalData"`
		// Labels IncidentLabel `json:"labels"`
	} `json:"properties"`
}

type ExpectedEvent struct {
	Event struct {
		Event struct {
			Metadata struct {
				GUID             int         `json:"guid"`
				Name             string      `json:"name"`
				URL              interface{} `json:"url"`
				Severity         string      `json:"severity"`
				ShortDescription string      `json:"shortDescription"`
				LongDescription  string      `json:"longDescription"`
				Time             int         `json:"time"`
			} `json:"metadata"`
			Producer struct {
				Name string `json:"name"`
			} `json:"producer"`
			Reporter struct {
				Name string `json:"name"`
			} `json:"reporter"`
			Resources []struct {
				GUID      string `json:"guid"`
				Name      string `json:"name"`
				Region    string `json:"region"`
				Platform  string `json:"platform"`
				Service   string `json:"service"`
				Type      string `json:"type"`
				AccountID string `json:"accountId"`
				Package   string `json:"package"`
			} `json:"resources"`
		} `json:"event"`
		Decoration []struct {
			Decorator string    `json:"decorator"`
			Timestamp time.Time `json:"timestamp"`
			Payload   struct {
				Registry         string    `json:"registry"`
				Namespace        string    `json:"namespace"`
				Image            string    `json:"image"`
				Tag              string    `json:"tag"`
				Digests          []string  `json:"digests"`
				ImageLastUpdated time.Time `json:"imageLastUpdated"`
				TagLastUpdated   time.Time `json:"tagLastUpdated"`
				Description      string    `json:"description"`
				StarCount        int       `json:"starCount"`
				PullCount        int64     `json:"pullCount"`
			} `json:"payload"`
		} `json:"decoration"`
	} `json:"event"`
	Sourcetype string `json:"sourcetype"`
}
