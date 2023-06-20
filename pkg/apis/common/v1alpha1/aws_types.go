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
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgapis "knative.dev/pkg/apis"

	"github.com/triggermesh/triggermesh/pkg/apis"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

// AnnotationEksIAMRole is the SA annotation used on EKS for IAM authentication.
const AnnotationEksIAMRole = "eks.amazonaws.com/role-arn"

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

	// Deprecation warning: please use IAM object instead.
	// +optional
	EksIAMRole *apis.ARN `json:"iamRole,omitempty"`

	// The IAM role authentication parameters. For Amazon EKS only.
	// +optional
	IAM *EksIAM `json:"iam,omitempty"`
}

// EksIAM contains parameters used for IAM authentication on EKS.
//
// +k8s:deepcopy-gen=true
type EksIAM struct {
	// The ARN of an IAM role which can be impersonated
	// to obtain AWS permissions.
	// See https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html
	// +optional
	Role *apis.ARN `json:"roleArn,omitempty"`

	// The name of the service account to be assigned on the receiver adapter.
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
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

// WantsOwnServiceAccount indicates wether the object requires its own SA.
func (a *AWSAuth) WantsOwnServiceAccount() bool {
	return a.EksIAMRole != nil || a.IAM != nil
}

// ServiceAccountOptions returns the set of SA mutations based on the object spec.
func (a *AWSAuth) ServiceAccountOptions() []resource.ServiceAccountOption {
	var saOpts []resource.ServiceAccountOption

	if a.EksIAMRole != nil {
		saOpts = append(saOpts, awsIamRoleAnnotation(*a.EksIAMRole))
	}
	if a.IAM == nil {
		return saOpts
	}
	if a.IAM.Role != nil {
		saOpts = append(saOpts, awsIamRoleAnnotation(*a.IAM.Role))
	}
	if a.IAM.ServiceAccount != "" {
		saOpts = append(saOpts, saName(a.IAM.ServiceAccount))
	}
	return saOpts
}

// awsIamRoleAnnotation returns a functional option that sets the EKS role-arn
// annotation on a ServiceAccount.
func awsIamRoleAnnotation(iamRole apis.ARN) resource.ServiceAccountOption {
	return func(sa *corev1.ServiceAccount) {
		metav1.SetMetaDataAnnotation(&sa.ObjectMeta, AnnotationEksIAMRole, iamRole.String())
	}
}

// saName returns a functional option that overwrites the
// Kubernetes Service Account name.
func saName(name string) resource.ServiceAccountOption {
	return func(sa *corev1.ServiceAccount) {
		sa.SetName(name)
	}
}

// Validate method is used to validate AWS objects' Auth spec.
func (a *AWSAuth) Validate(ctx context.Context) *pkgapis.FieldError {
	if a.EksIAMRole != nil {
		return pkgapis.ErrDisallowedUpdateDeprecatedFields("auth.iamRole")
	}
	return nil
}
