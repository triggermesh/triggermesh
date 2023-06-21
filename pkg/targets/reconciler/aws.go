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

package reconciler

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler"
)

// MakeAWSAuthEnvVars returns environment variables for the given AWS
// authentication method.
func MakeAWSAuthEnvVars(auth v1alpha1.AWSAuth) []corev1.EnvVar {
	var authEnvVars []corev1.EnvVar

	if creds := auth.Credentials; creds != nil {
		authEnvVars = reconciler.MaybeAppendValueFromEnvVar(authEnvVars, reconciler.EnvAccessKeyID, creds.AccessKeyID)
		authEnvVars = reconciler.MaybeAppendValueFromEnvVar(authEnvVars, reconciler.EnvSecretAccessKey, creds.SecretAccessKey)
		authEnvVars = reconciler.MaybeAppendValueFromEnvVar(authEnvVars, reconciler.EnvSessionToken, creds.SessionToken)

		if creds.AssumeIAMRole != nil {
			authEnvVars = append(authEnvVars, corev1.EnvVar{
				Name:  reconciler.EnvAssumeIamRole,
				Value: creds.AssumeIAMRole.String(),
			})
		}
	}

	return authEnvVars
}
