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

package confluenttarget

import (
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	// This block of environment variables are expected to be informed
	// by the the Confluent object and as such part of the CRD.
	BootstrapServers string `envconfig:"CONFLUENT_BOOTSTRAP_SERVERS" required:"true"`
	SASLUsername     string `envconfig:"CONFLUENT_SASL_USERNAME" required:"true"`
	SASLPassword     string `envconfig:"CONFLUENT_SASL_PASSWORD" required:"true"`
	Topic            string `envconfig:"CONFLUENT_TOPIC" required:"true"`
	SASLMechanisms   string `envconfig:"CONFLUENT_SASL_MECHANISMS" required:"false" default:"PLAIN"`
	SecurityProtocol string `envconfig:"CONFLUENT_SECURITY_PROTOCOL" required:"false" default:"SASL_SSL"`

	// This set of variables are experimental and not graduated to the CRD.
	BrokerVersionFallback       string `envconfig:"CONFLUENT_BROKER_VERSION_FALLBACK" required:"false" default:"0.10.0.0"`
	APIVersionFallbackMs        string `envconfig:"CONFLUENT_API_VERSION_FALLBACK_MS" required:"false" default:"0"`
	CreateTopicIfMissing        bool   `envconfig:"CONFLUENT_CREATE_MISSING_TOPIC" default:"true"`
	FlushOnExitTimeoutMillisecs int    `envconfig:"CONFLUENT_FLUSH_ON_EXIT_TIMEOUT_MS" default:"10000"`
	CreateTopicTimeoutMillisecs int    `envconfig:"CONFLUENT_CREATE_TOPIC_TIMEOUT_MS" default:"10000"`
	NewTopicPartitions          int    `envconfig:"CONFLUENT_TOPIC_PARTITIONS" default:"1"`
	NewTopicReplicationFactor   int    `envconfig:"CONFLUENT_TOPIC_REPLICATION_FACTOR" default:"1"`

	DiscardCEContext bool `envconfig:"CONFLUENT_DISCARD_CE_CONTEXT"`
}
