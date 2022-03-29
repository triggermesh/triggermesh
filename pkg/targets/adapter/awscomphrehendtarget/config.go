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

package awscomphrehendtarget

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

	AWSApiKey string `envconfig:"AWS_ACCESS_KEY_ID" required:"true"`

	AWSApiSecret string `envconfig:"AWS_SECRET_ACCESS_KEY" required:"true"`

	Region string `envconfig:"COMPREHEND_REGION" required:"true"`

	Language string `envconfig:"COMPREHEND_LANGUAGE" required:"true"`

	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`

	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
}

func (e *envAccessor) GetAwsConfig(region string) *aws.Config {
	creds := credentials.NewStaticCredentials(e.AWSApiKey, e.AWSApiSecret, "")
	config := aws.NewConfig()
	config.WithRegion(region)
	config.WithCredentials(creds)

	return config
}
