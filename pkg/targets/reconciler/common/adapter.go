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

package common

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

// MaybeAppendValueFromEnvVar conditionally appends an EnvVar to env based on
// the contents of valueFrom.
// ValueFromSecret takes precedence over Value in case the API didn't reject
// the object despite the CRD's schema validation
func MaybeAppendValueFromEnvVar(envs []corev1.EnvVar, key string, valueFrom v1alpha1.ValueFromField) []corev1.EnvVar {
	if vfs := valueFrom.ValueFromSecret; vfs != nil {
		return append(envs, corev1.EnvVar{
			Name: key,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: vfs,
			},
		})
	}

	if v := valueFrom.Value; v != "" {
		return append(envs, corev1.EnvVar{
			Name:  key,
			Value: v,
		})
	}

	return envs
}
