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

package awskinesistarget

import (
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// NewEnvConfig for configuration parameters
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	AwsTargetArn        string `envconfig:"ARN" required:"true"`
	AwsKinesisPartition string `envconfig:"AWS_KINESIS_PARTITION"`

	DiscardCEContext bool `envconfig:"AWS_DISCARD_CE_CONTEXT"`

	// Assume this IAM Role when access keys provided.
	AssumeIamRole string `envconfig:"AWS_ASSUME_ROLE_ARN"`

	// The environment variables below aren't read from the envConfig struct
	// by the AWS SDK, but rather directly using os.Getenv().
	// They are nevertheless listed here for documentation purposes.
	_ string `envconfig:"AWS_ACCESS_KEY_ID"`
	_ string `envconfig:"AWS_SECRET_ACCESS_KEY"`
}
