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
	pkgapis "knative.dev/pkg/apis"

	"github.com/triggermesh/triggermesh/pkg/apis"
	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// AWSAuth contains multiple authentication methods for AWS services.
type AWSAuth struct {
	// Security credentials allow AWS to authenticate and authorize
	// requests based on a signature composed of an access key ID and a
	// corresponding secret access key.
	// See https://docs.aws.amazon.com/general/latest/gr/aws-security-credentials.html
	Credentials *AWSSecurityCredentials `json:"credentials,omitempty"`

	// (Amazon EKS only) The ARN of an IAM role which can be impersonated
	// to obtain AWS permissions.
	// See https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html
	EksIAMRole *apis.ARN `json:"iamRole"`
}

// AWSSecurityCredentials represents a set of AWS security credentials.
type AWSSecurityCredentials struct {
	AccessKeyID     v1alpha1.ValueFromField `json:"accessKeyID"`
	SecretAccessKey v1alpha1.ValueFromField `json:"secretAccessKey"`
}

// AWSEndpoint contains parameters which are used to override the destination
// of REST API calls to AWS services.
// It allows, for example, to target API-compatible alternatives to the public
// AWS cloud (Localstack, Minio, ElasticMQ, ...).
type AWSEndpoint struct {
	// URL of the endpoint.
	URL *pkgapis.URL `json:"url,omitempty"`
}
