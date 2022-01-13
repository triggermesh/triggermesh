//go:build !noclibs

/*
Copyright 2021 TriggerMesh Inc.

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

package ibmmqsource

import (
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/ibmmqsource/mq"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

var _ pkgadapter.EnvConfigAccessor = (*SourceEnvAccessor)(nil)

// SourceDeliveryConfig holds the delivery parameters used in the source.
type SourceDeliveryConfig struct {
	DeadLetterQManager string `envconfig:"DEAD_LETTER_QUEUE_MANAGER"`
	DeadLetterQueue    string `envconfig:"DEAD_LETTER_QUEUE"`
	BackoffDelay       int    `envconfig:"BACKOFF_DELAY"`
	Retry              int    `envconfig:"DELIVERY_RETRY"`
}

// SourceEnvAccessor is the set of parameters parsed from the adapter's env.
type SourceEnvAccessor struct {
	pkgadapter.EnvConfig
	mq.EnvConnectionConfig
	SourceDeliveryConfig

	// BridgeIdentifier is the name of the bridge workflow this source is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
}

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &SourceEnvAccessor{}
}

// Delivery returns the MQ delivery parameters.
func (e *SourceEnvAccessor) Delivery() *mq.Delivery {
	if e.DeadLetterQManager == "" {
		e.DeadLetterQManager = e.QueueManager
	}
	return &mq.Delivery{
		DeadLetterQManager: e.DeadLetterQManager,
		DeadLetterQueue:    e.DeadLetterQueue,
		BackoffDelay:       e.BackoffDelay,
		Retry:              e.Retry,
	}
}
