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

package resource

import (
	"path"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// PodLabel sets the value of a label of a PodSpecable's Pod template.
func PodLabel(key, val string) ObjectOption {
	return func(object interface{}) {
		var metaObj metav1.Object

		switch o := object.(type) {
		case *appsv1.Deployment:
			metaObj = &o.Spec.Template
		case *servingv1.Service:
			metaObj = &o.Spec.Template
		}

		Label(key, val)(metaObj)
	}
}

// Container adds a container to a PodSpecable's Pod template.
func Container(c *corev1.Container) ObjectOption {
	return func(object interface{}) {
		switch o := object.(type) {
		case *appsv1.Deployment:
			containers := &o.Spec.Template.Spec.Containers
			*containers = append(*containers, *c)
		case *servingv1.Service:
			containers := &o.Spec.Template.Spec.Containers
			*containers = []corev1.Container{*c}
		}
	}
}

// ServiceAccount sets the ServiceAccount name of a PodSpecable.
func ServiceAccount(sa string) ObjectOption {
	return func(object interface{}) {
		var saName *string

		switch o := object.(type) {
		case *appsv1.Deployment:
			saName = &o.Spec.Template.Spec.ServiceAccountName
		case *servingv1.Service:
			saName = &o.Spec.Template.Spec.ServiceAccountName
		}

		*saName = sa
	}
}

// SecretMount adds a Secret volume and a corresponding mount to a PodSpecable.
func SecretMount(name, target, secretName, secretKey string) ObjectOption {
	return func(object interface{}) {
		var vols *[]corev1.Volume

		switch o := object.(type) {
		case *appsv1.Deployment:
			vols = &o.Spec.Template.Spec.Volumes
		case *servingv1.Service:
			vols = &o.Spec.Template.Spec.Volumes
		}

		*vols = append(*vols, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
					Items: []corev1.KeyToPath{
						{
							Key:  secretKey,
							Path: path.Base(target),
						},
					},
				},
			},
		})

		var volMounts *[]corev1.VolumeMount

		switch o := object.(type) {
		case *appsv1.Deployment, *servingv1.Service:
			volMounts = &firstContainer(o).VolumeMounts
		}

		*volMounts = append(*volMounts, corev1.VolumeMount{
			Name:      name,
			ReadOnly:  true,
			MountPath: target,
			SubPath:   path.Base(target),
		})
	}
}
