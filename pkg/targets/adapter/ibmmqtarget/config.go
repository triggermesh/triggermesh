//go:build !noclibs

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

package ibmmqtarget

import (
	"github.com/triggermesh/triggermesh/pkg/targets/adapter/ibmmqtarget/mq"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

var _ pkgadapter.EnvConfigAccessor = (*TargetEnvAccessor)(nil)

// TargetEnvAccessor is the set of parameters parsed from the adapter's env.
type TargetEnvAccessor struct {
	pkgadapter.EnvConfig
	mq.ConnectionConfig
	mq.ReplyTo
	mq.Auth

	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
	DiscardCEContext        bool   `envconfig:"DISCARD_CE_CONTEXT"`

	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
}

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &TargetEnvAccessor{}
}
