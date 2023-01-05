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

package kafkatarget

import (
	"time"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	SASLEnable bool `envconfig:"SASL_ENABLE" required:"false"`
	TLSEnable  bool `envconfig:"TLS_ENABLE" required:"false"`

	BootstrapServers   []string `envconfig:"BOOTSTRAP_SERVERS" required:"true"`
	Username           string   `envconfig:"USERNAME" required:"false"`
	Password           string   `envconfig:"PASSWORD" required:"false"`
	Topic              string   `envconfig:"TOPIC" required:"true"`
	GroupID            string   `envconfig:"GROUP_ID" required:"false" `
	SecurityMechanisms string   `envconfig:"SECURITY_MECHANISMS" required:"false"`

	KerberosConfigPath  string `envconfig:"KERBEROS_CONFIG_PATH" required:"false" `
	KerberosServiceName string `envconfig:"KERBEROS_SERVICE_NAME" required:"false" `
	KerberosKeytabPath  string `envconfig:"KERBEROS_KEYTAB_PATH" required:"false"`
	KerberosRealm       string `envconfig:"KERBEROS_REALM" required:"false"`
	KerberosUsername    string `envconfig:"KERBEROS_USERNAME" required:"false"`
	KerberosPassword    string `envconfig:"KERBEROS_PASSWORD" required:"false"`

	CA         string `envconfig:"CA" required:"false"`
	ClientCert string `envconfig:"CLIENT_CERT" required:"false"`
	ClientKey  string `envconfig:"CLIENT_KEY" required:"false"`
	SkipVerify bool   `envconfig:"SKIP_VERIFY" required:"false"`

	// The connection refresh routine will discard the existing connection and create a new
	// one. The Kafka broker is usually configured to close idle connections at its side after 10 minutes,
	// 5 minutes is a safe guess for most instances.
	ConnectionRefreshPeriod time.Duration `envconfig:"CONNECTION_REFRESH_PERIOD" default:"5m"`

	// This set of variables are experimental and not graduated to the CRD.
	CreateTopicIfMissing        bool  `envconfig:"CREATE_MISSING_TOPIC" default:"true"`
	FlushOnExitTimeoutMillisecs int   `envconfig:"FLUSH_ON_EXIT_TIMEOUT_MS" default:"10000"`
	CreateTopicTimeoutMillisecs int   `envconfig:"CREATE_TOPIC_TIMEOUT_MS" default:"10000"`
	NewTopicPartitions          int32 `envconfig:"TOPIC_PARTITIONS" default:"1"`
	NewTopicReplicationFactor   int16 `envconfig:"TOPIC_REPLICATION_FACTOR" default:"1"`

	DiscardCEContext bool `envconfig:"DISCARD_CE_CONTEXT"`
}
