/*
Copyright 2023 TriggerMesh Inc.

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

package solacesource

import (
	"knative.dev/eventing/pkg/adapter/v2"
)

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() adapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	adapter.EnvConfig

	SASLEnable bool `envconfig:"SASL_ENABLE" required:"false"`
	TLSEnable  bool `envconfig:"TLS_ENABLE" required:"false"`

	URL       string `envconfig:"URL" required:"true"`
	QueueName string `envconfig:"QUEUE_NAME" required:"true"`
	Username  string `envconfig:"USERNAME" required:"false"`
	Password  string `envconfig:"PASSWORD" required:"false"`

	CA         string `envconfig:"CA" required:"false"`
	ClientCert string `envconfig:"CLIENT_CERT" required:"false"`
	ClientKey  string `envconfig:"CLIENT_KEY" required:"false"`
	SkipVerify bool   `envconfig:"SKIP_VERIFY" required:"false"`
}
