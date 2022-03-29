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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewDeployment creates a Deployment object.
func NewDeployment(ns, name string, opts ...ObjectOption) *appsv1.Deployment {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}

	for _, opt := range opts {
		opt(d)
	}

	// If the Deployment was created without defining a Container
	// explicitly, ensure its default container's name is not empty.
	containers := d.Spec.Template.Spec.Containers
	if len(containers) == 1 && containers[0].Name == "" {
		containers[0].Name = defaultContainerName
	}

	return d
}

// Selector adds a label selector to a Deployment's spec, ensuring a
// corresponding label exists in the Pod template.
func Selector(key, val string) ObjectOption {
	return func(object interface{}) {
		d := object.(*appsv1.Deployment)

		selector := &d.Spec.Selector

		if *selector == nil {
			*selector = &metav1.LabelSelector{}
		}
		*selector = metav1.AddLabelToSelector(*selector, key, val)

		PodLabel(key, val)(d)
	}
}
