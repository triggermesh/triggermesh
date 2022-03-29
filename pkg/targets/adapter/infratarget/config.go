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

package infratarget

import (
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	ScriptCode         string `envconfig:"INFRA_SCRIPT_CODE" required:"true"`
	ScriptTimeout      int    `envconfig:"INFRA_SCRIPT_TIMEOUT" default:"2000"`
	StateHeadersPolicy string `envconfig:"INFRA_STATE_HEADERS_POLICY" default:"propagate"`
	StateBridge        string `envconfig:"INFRA_STATE_BRIDGE"`
	TypeLoopProtection bool   `envconfig:"INFRA_TYPE_LOOP_PROTECTION" default:"true"`
}
