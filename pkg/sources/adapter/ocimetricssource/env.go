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

package ocimetricssource

import (
	"knative.dev/eventing/pkg/adapter/v2"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() adapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	adapter.EnvConfig

	OracleAPIKey            string `envconfig:"ORACLE_API_PRIVATE_KEY" required:"true"`
	OracleAPIKeyPassphrase  string `envconfig:"ORACLE_API_PRIVATE_KEY_PASSPHRASE" required:"true"`
	OracleAPIKeyFingerprint string `envconfig:"ORACLE_API_PRIVATE_KEY_FINGERPRINT" required:"true"`
	UserOCID                string `envconfig:"ORACLE_USER_OCID" required:"true"`
	TenantOCID              string `envconfig:"ORACLE_TENANT_OCID" required:"true"`
	OracleRegion            string `envconfig:"ORACLE_REGION" required:"true"`

	PollingFrequency string                         `envconfig:"ORACLE_METRICS_POLLING_FREQUENCY" required:"true"`
	Metrics          v1alpha1.OCIMetricsDecodedList `envconfig:"ORACLE_METRICS" required:"true"`
}
