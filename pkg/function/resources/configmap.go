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

package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"
)

const DefaultCmKey = "code"

// Option sets configmap options
type cmOption func(*corev1.ConfigMap)

func NewConfigmap(name, namespace string, opts ...cmOption) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	for _, opt := range opts {
		opt(cm)
	}

	return cm
}

func CmData(data string) cmOption {
	return func(cm *corev1.ConfigMap) {
		cm.Data = map[string]string{
			DefaultCmKey: data,
		}
	}
}

func CmOwner(o kmeta.OwnerRefable) cmOption {
	return func(cm *corev1.ConfigMap) {
		cm.SetOwnerReferences([]metav1.OwnerReference{
			*kmeta.NewControllerRef(o),
		})
	}
}

func CmLabel(labels map[string]string) cmOption {
	return func(cm *corev1.ConfigMap) {
		cm.SetLabels(labels)
	}
}
