/*
Copyright 2020 TriggerMesh Inc.

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

package cloudeventssource

import (
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"knative.dev/eventing/pkg/adapter/v2"
)

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() adapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	adapter.EnvConfig

	Path           string                   `envconfig:"CLOUDEVENTS_PATH"`
	RateLimiterRPS int64                    `envconfig:"CLOUDEVENTS_RATELIMITER_RPS"`
	BasicAuths     []v1alpha1.HTTPBasicAuth `envconfig:"CLOUDEVENTS_BASICAUTH_CREDENTIALS"`
	Tokens         []v1alpha1.HTTPToken     `envconfig:"CLOUDEVENTS_TOKEN_CREDENTIALS"`
	// CORSAllowOrigin string                   `envconfig:"CLOUDEVENTS_CORS_ALLOW_ORIGIN"`
}
