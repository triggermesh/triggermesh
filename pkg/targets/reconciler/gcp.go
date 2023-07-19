/*
Copyright 2023 TriggerMesh Inc.

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
	targetsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
)

// MakeGCPAuthEnvVars accepts both old credentials and new auth object and
// returns adapter environment with configured authentication variables.
func MakeGCPAuthEnvVars(creds *targetsv1alpha1.SecretValueFromSource, auth *v1alpha1.GoogleCloudAuth) []corev1.EnvVar {
	var env []corev1.EnvVar

	if creds != nil && creds.SecretKeyRef != nil {
		env = append(env, corev1.EnvVar{
			Name: common.EnvGCloudSAKey,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: creds.SecretKeyRef,
			},
		})
	}
	if auth != nil && auth.ServiceAccountKey != nil {
		env = common.MaybeAppendValueFromEnvVar(env, common.EnvGCloudSAKey, *auth.ServiceAccountKey)
	}

	return env
}
