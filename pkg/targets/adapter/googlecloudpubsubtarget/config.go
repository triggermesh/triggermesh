package googlecloudpubsubtarget

import (
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/googlecloudpubsubsource"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	TopicName googlecloudpubsubsource.GCloudResourceName `envconfig:"GCLOUD_PUBSUB_TOPIC" required:"true"`

	ServiceAccountKey []byte `envconfig:"GCLOUD_SERVICEACCOUNT_KEY" required:"true"`

	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
}
