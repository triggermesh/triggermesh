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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgapis "knative.dev/pkg/apis"

	"github.com/triggermesh/triggermesh/pkg/apis"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

// AWSAuth contains multiple authentication methods for AWS services.
//
// +k8s:deepcopy-gen=true
type AWSAuth struct {
	// Security credentials allow AWS to authenticate and authorize
	// requests based on a signature composed of an access key ID and a
	// corresponding secret access key.
	// See https://docs.aws.amazon.com/general/latest/gr/aws-security-credentials.html
	// +optional
	Credentials *AWSSecurityCredentials `json:"credentials,omitempty"`

	// (Amazon EKS only) The ARN of an IAM role which can be impersonated
	// to obtain AWS permissions.
	// See https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html
	// +optional
	EksIAMRole *apis.ARN `json:"iamRole,omitempty"`
}

// AWSSecurityCredentials represents a set of AWS security credentials.
//
// +k8s:deepcopy-gen=true
type AWSSecurityCredentials struct {
	AccessKeyID     ValueFromField `json:"accessKeyID"`
	SecretAccessKey ValueFromField `json:"secretAccessKey"`

	// The AWS session token for temporary credentials.
	// See https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_request.html#api_getsessiontoken
	// +optional
	SessionToken ValueFromField `json:"sessionToken,omitempty"`

	// The ARN of an IAM role for cross-account or remote impersonation on EKS.
	// Require the access key credentials to create a client session.
	// +optional
	AssumeIAMRole *apis.ARN `json:"assumeIamRole,omitempty"`
}

// AWSEndpoint contains parameters which are used to override the destination
// of REST API calls to AWS services.
// It allows, for example, to target API-compatible alternatives to the public
// AWS cloud (Localstack, Minio, ElasticMQ, ...).
//
// +k8s:deepcopy-gen=true
type AWSEndpoint struct {
	// URL of the endpoint.
	URL *pkgapis.URL `json:"url,omitempty"`
}

const annotationEksIAMRole = "eks.amazonaws.com/role-arn"

// AwsIamRoleAnnotation returns a functional option that sets the EKS role-arn
// annotation on a ServiceAccount.
func AwsIamRoleAnnotation(iamRole apis.ARN) resource.ServiceAccountOption {
	return func(sa *corev1.ServiceAccount) {
		metav1.SetMetaDataAnnotation(&sa.ObjectMeta, annotationEksIAMRole, iamRole.String())
	}
}
