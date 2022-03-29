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

// Package env allows propagating runtime configurations via the environment.
package env

import (
	"github.com/kelseyhightower/envconfig"
	"knative.dev/eventing/pkg/adapter/v2"
)

// ConfigAccessor is a superset of adaper.EnvConfigAccessor that overrides
// properties about certain variables.
type ConfigAccessor interface {
	adapter.EnvConfigAccessor
	// Get the component name.
	GetComponent() string
}

// Config is the minimal set of configuration parameters routing adapters should support.
type Config struct {
	*adapter.EnvConfig
	// Environment variable containing the namespace of the adapter.
	Namespace string `envconfig:"NAMESPACE" required:"true"`
	// Component is the kind of this adapter.
	Component string `envconfig:"K_COMPONENT" required:"true"`
}

// Verify that Config implements ConfigAccessor.
var _ ConfigAccessor = (*Config)(nil)

// GetComponent implements ConfigAccessor.
func (c *Config) GetComponent() string {
	return c.Component
}

// ConfigConstructor is a callback function that returns a ConfigAccessor.
type ConfigConstructor func() ConfigAccessor

// MustProcessConfig populates the specified adapter.EnvConfigConstructor based
// on environment variables.
func MustProcessConfig(envCtor ConfigConstructor) ConfigAccessor {
	env := envCtor()
	envconfig.MustProcess("", env)
	return env
}
