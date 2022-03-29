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

package salesforcetarget

import (
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessor for configuration parameters
func EnvAccessor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	ClientID   string `envconfig:"SALESFORCE_AUTH_CLIENT_ID" required:"true"`
	AuthServer string `envconfig:"SALESFORCE_AUTH_SERVER" required:"true"`
	User       string `envconfig:"SALESFORCE_AUTH_USER" required:"true"`
	CertKey    string `envconfig:"SALESFORCE_AUTH_CERT_KEY" required:"true"`
	Version    string `envconfig:"SALESFORCE_API_VERSION"`

	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"always"`

	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
}
