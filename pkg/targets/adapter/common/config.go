package common

import (
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

type CommonEnv struct {
	pkgadapter.EnvConfig

	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`

	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
	DiscardCEContext        bool   `envconfig:"DISCARD_CE_CONTEXT" default:"false"`

	// EventTransportLayer is the name of the transport layer used to send events
	// options are: "CE", or "NKN".
	EventTransportLayer string `envconfig:"EVENTS_TRANSPORT_LAYER" default:"CE"`

	// Optional component parameters required by the NKN transport layer
	// When using the CE transport layer, these parameters are ignored/not required.
	Seed string `envconfig:"EVENTS_NKN_SEED"`
}
