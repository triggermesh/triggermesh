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

import corev1 "k8s.io/api/core/v1"

var (
	tValue = "test-value"
	tName  = "test-name"
	tKey   = "test-key"

	tEnvName = "TEST_ENV"
)

type valueFromFieldOptions func(*ValueFromField)

func valueFromField(opts ...valueFromFieldOptions) *ValueFromField {
	vff := &ValueFromField{}

	for _, o := range opts {
		o(vff)
	}
	return vff
}

func vffWithValue(value string) valueFromFieldOptions {
	return func(vff *ValueFromField) {
		vff.Value = value
	}
}

func vffWithSecret(name, key string) valueFromFieldOptions {
	return func(vff *ValueFromField) {
		vff.ValueFromSecret = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
			Key: key,
		}
	}
}

func vffWithConfigMap(name, key string) valueFromFieldOptions {
	return func(vff *ValueFromField) {
		vff.ValueFromConfigMap = &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
			Key: key,
		}
	}
}

type envVarFromFieldOptions func(*corev1.EnvVar)

func envVar(opts ...envVarFromFieldOptions) *corev1.EnvVar {
	ev := &corev1.EnvVar{}

	for _, o := range opts {
		o(ev)
	}
	return ev
}

func evWithValue(value string) envVarFromFieldOptions {
	return func(ev *corev1.EnvVar) {
		ev.Value = value
	}
}

func evWithSecret(name, key string) envVarFromFieldOptions {
	return func(ev *corev1.EnvVar) {
		ev.ValueFrom = &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				Key: key,
			},
		}
	}
}

func evWithConfigMap(name, key string) envVarFromFieldOptions {
	return func(ev *corev1.EnvVar) {
		ev.ValueFrom = &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				Key: key,
			},
		}
	}
}
