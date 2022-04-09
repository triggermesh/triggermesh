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

package twiliotarget

import (
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig
	// AccountSID is the Twilio account SID
	AccountSID string `envconfig:"TWILIO_SID" required:"true"`
	// Token is the API key for the Twilio account.
	Token string `envconfig:"TWILIO_TOKEN" required:"true"`
	// PhoneFrom is the phone number to use as the sender of the SMS
	PhoneFrom string `envconfig:"TWILIO_DEFAULT_FROM" required:"false"`
	// PhoneTo is the phone number to send the message to
	PhoneTo string `envconfig:"TWILIO_DEFAULT_TO" required:"false"`

	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"always"`
	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
}
