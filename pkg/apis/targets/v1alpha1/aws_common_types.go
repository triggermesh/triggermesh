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

package v1alpha1

import (
	"github.com/triggermesh/triggermesh/pkg/apis"
)

// TODO: migrate CRDs to auth structure from
// pkg/apis/sources/v1alpha1/aws_common_types.go

// AWSAuth contains multiple authentication methods for AWS services.
type AWSAuth struct {
	// (Amazon EKS only) The ARN of an IAM role which can be impersonated
	// to obtain AWS permissions.
	// See https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html
	// +optional
	EksIAMRole *apis.ARN `json:"iamRole,omitempty"`

	// AWS account Key.
	AWSApiKey SecretValueFromSource `json:"awsApiKey"`

	// AWS account secret key.
	AWSApiSecret SecretValueFromSource `json:"awsApiSecret"`
}
