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

package sendgridtarget

import (
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig
	APIKey    string `envconfig:"SENDGRID_API_KEY" required:"true"`
	FromEmail string `envconfig:"SENDGRID_DEFAULT_FROM_EMAIL" required:"false"`
	ToEmail   string `envconfig:"SENDGRID_DEFAULT_TO_EMAIL" required:"false"`
	FromName  string `envconfig:"SENDGRID_DEFAULT_FROM_NAME" required:"false"`
	ToName    string `envconfig:"SENDGRID_DEFAULT_TO_NAME" required:"false"`
	Message   string `envconfig:"SENDGRID_DEFAULT_MESSAGE" required:"false"`
	Subject   string `envconfig:"SENDGRID_DEFAULT_SUBJECT" required:"false"`

	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`

	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
}
