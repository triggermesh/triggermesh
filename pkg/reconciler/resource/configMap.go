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

package resource

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewConfigMap creates a ConfigMap object.
func NewConfigMap(ns, name string, opts ...ObjectOption) *corev1.ConfigMap {
	cmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}

	for _, opt := range opts {
		opt(cmap)
	}

	return cmap
}

// Data sets one UTF-8 data entry in a ConfigMap.
func Data(key, value string) ObjectOption {
	return func(object interface{}) {
		cmap := object.(*corev1.ConfigMap)

		bdata := &cmap.Data

		if *bdata == nil {
			*bdata = make(map[string]string, 1)
		}

		(*bdata)[key] = value
	}
}
