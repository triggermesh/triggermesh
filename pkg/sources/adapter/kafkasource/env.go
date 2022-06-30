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

package kafkasource

import (
	"knative.dev/eventing/pkg/adapter/v2"
)

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() adapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	adapter.EnvConfig

	SASLEnable bool `envconfig:"SASL_ENABLE" required:"true"`
	TLSEnable  bool `envconfig:"TLS_ENABLE" required:"true"`

	BootstrapServers []string `envconfig:"BOOTSTRAP_SERVERS" required:"true"`
	Username         string   `envconfig:"USERNAME" required:"false"`
	Password         string   `envconfig:"PASSWORD" required:"false"`
	Topics           []string `envconfig:"TOPICS" required:"true"`
	GroupID          string   `envconfig:"GROUP_ID" required:"false"`

	SecurityMechanisms  string `envconfig:"SECURITY_MECHANISMS" required:"true"`
	KerberosConfigPath  string `envconfig:"KERBEROS_CONFIG_PATH" required:"false" `
	KerberosServiceName string `envconfig:"KERBEROS_SERVICE_NAME" required:"false" `
	KerberosKeytabPath  string `envconfig:"KERBEROS_KEYTAB_PATH" required:"false"`
	KerberosRealm       string `envconfig:"KERBEROS_REALM" required:"false"`
	KerberosUsername    string `envconfig:"KERBEROS_USERNAME" required:"false"`
	KerberosPassword    string `envconfig:"KERBEROS_PASSWORD" required:"false"`

	SSLCA                 string `envconfig:"SSL_CA" required:"false"`
	SSLClientCert         string `envconfig:"SSL_CLIENT_CERT" required:"false"`
	SSLClientKey          string `envconfig:"SSL_CLIENT_KEY" required:"false"`
	SSLInsecureSkipVerify bool   `envconfig:"SSL_INSECURE_SKIP_VERIFY" required:"false"`

	// This set of variables are experimental and not graduated to the CRD.
	BrokerVersionFallback string `envconfig:"BROKER_VERSION_FALLBACK" required:"false" default:"0.10.0.0"`
	APIVersionFallbackMs  string `envconfig:"API_VERSION_FALLBACK_MS" required:"false" default:"0"`
}
