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

package httppollersource

import (
	"time"

	"knative.dev/eventing/pkg/adapter/v2"
)

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() adapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	adapter.EnvConfig

	EventType         string            `envconfig:"HTTPPOLLER_EVENT_TYPE" required:"true"`
	EventSource       string            `envconfig:"HTTPPOLLER_EVENT_SOURCE" required:"true"`
	Endpoint          string            `envconfig:"HTTPPOLLER_ENDPOINT" required:"true"`
	Method            string            `envconfig:"HTTPPOLLER_METHOD" required:"true"`
	SkipVerify        bool              `envconfig:"HTTPPOLLER_SKIP_VERIFY"`
	CACertificate     string            `envconfig:"HTTPPOLLER_CA_CERTIFICATE"`
	BasicAuthUsername string            `envconfig:"HTTPPOLLER_BASICAUTH_USERNAME"`
	BasicAuthPassword string            `envconfig:"HTTPPOLLER_BASICAUTH_PASSWORD"`
	Headers           map[string]string `envconfig:"HTTPPOLLER_HEADERS"`
	Interval          time.Duration     `envconfig:"HTTPPOLLER_INTERVAL" required:"true"`
}
