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

package salesforcesource

import "knative.dev/eventing/pkg/adapter/v2"

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() adapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	adapter.EnvConfig

	ClientID   string `envconfig:"SALESFORCE_AUTH_CLIENT_ID" required:"true"`
	AuthServer string `envconfig:"SALESFORCE_AUTH_SERVER" required:"true"`
	User       string `envconfig:"SALESFORCE_AUTH_USER" required:"true"`
	CertKey    string `envconfig:"SALESFORCE_AUTH_CERT_KEY" required:"true"`
	Version    string `envconfig:"SALESFORCE_API_VERSION" default:"48.0"`

	// We are supporting only one subscription + replayID per source instance
	SubscriptionChannel  string `envconfig:"SALESFORCE_SUBCRIPTION_CHANNEL" required:"true"`
	SubscriptionReplayID int    `envconfig:"SALESFORCE_SUBCRIPTION_REPLAY_ID" default:"-1"`
}
