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

package webhooksource

import (
	"fmt"
	"strings"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	EventType                    string                   `envconfig:"WEBHOOK_EVENT_TYPE" required:"true"`
	EventSource                  string                   `envconfig:"WEBHOOK_EVENT_SOURCE" required:"true"`
	EventExtensionAttributesFrom *ExtensionAttributesFrom `envconfig:"WEBHOOK_EVENT_EXTENSION_ATTRIBUTES_FROM"`
	BasicAuthUsername            string                   `envconfig:"WEBHOOK_BASICAUTH_USERNAME"`
	BasicAuthPassword            string                   `envconfig:"WEBHOOK_BASICAUTH_PASSWORD"`
	CORSAllowOrigin              string                   `envconfig:"WEBHOOK_CORS_ALLOW_ORIGIN"`
}

type ExtensionAttributesFrom struct {
	method  bool
	path    bool
	host    bool
	queries bool
	headers bool
}

// Decode an array of KeyMountedValues
func (ea *ExtensionAttributesFrom) Decode(value string) error {
	for _, o := range strings.Split(value, ",") {
		switch o {
		case "method":
			ea.method = true
		case "path":
			ea.path = true
		case "host":
			ea.host = true
		case "queries":
			ea.queries = true
		case "headers":
			ea.headers = true
		default:
			return fmt.Errorf("CloudEvent extension from HTTP element not supported: %s", o)
		}
	}
	return nil
}
