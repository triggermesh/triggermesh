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

package awstarget

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	AWSApiKey           string `envconfig:"AWS_ACCESS_KEY_ID" required:"true"`
	AWSApiSecret        string `envconfig:"AWS_SECRET_ACCESS_KEY" required:"true"`
	AwsTargetType       string `envconfig:"AWS_TARGET_TYPE" required:"true"`
	AwsTargetArn        string `envconfig:"AWS_TARGET_ARN" required:"true"`
	AwsKinesisPartition string `envconfig:"AWS_KINESIS_PARTITION"`

	DiscardCEContext bool `envconfig:"AWS_DISCARD_CE_CONTEXT"`
}

func (e *envAccessor) GetAwsConfig() *aws.Config {
	creds := credentials.NewStaticCredentials(e.AWSApiKey, e.AWSApiSecret, "")
	config := aws.NewConfig()

	config.WithCredentials(creds)

	return config
}
