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

package cloudeventssource

import (
	"encoding/json"

	cereconciler "github.com/triggermesh/triggermesh/pkg/sources/reconciler/cloudeventssource"
	"knative.dev/eventing/pkg/adapter/v2"
)

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() adapter.EnvConfigAccessor {
	return &envAccessor{}
}

// KeyMountedValues contains a set of file mounted values
// by their name.
type KeyMountedValues []cereconciler.KeyMountedValue

// Decode an array of KeyMountedValues
func (is *KeyMountedValues) Decode(value string) error {
	if err := json.Unmarshal([]byte(value), is); err != nil {
		return err
	}
	return nil
}

type envAccessor struct {
	adapter.EnvConfig

	Path              string           `envconfig:"CLOUDEVENTS_PATH"`
	BasicAuths        KeyMountedValues `envconfig:"CLOUDEVENTS_BASICAUTH_CREDENTIALS"`
	RequestsPerSecond uint64           `envconfig:"CLOUDEVENTS_RATELIMITER_RPS"`
}
