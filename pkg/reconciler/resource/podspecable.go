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

// SecretMountOption is an option function to customize secret volume mounts.
type SecretMountOption func(v *corev1.Volume, vm *corev1.VolumeMount)

// WithMountSubPath modifies a secret volume mount to use a subpath.
func WithMountSubPath(subpath string) SecretMountOption {
	return func(_ *corev1.Volume, vm *corev1.VolumeMount) {
		vm.SubPath = subpath
	}
}

// WithVolumeSecretItem modifies a secret volume to add a
// selected key and path.
func WithVolumeSecretItem(key, path string) SecretMountOption {
	return func(v *corev1.Volume, _ *corev1.VolumeMount) {
		v.VolumeSource.Secret.Items = append(
			v.VolumeSource.Secret.Items,
			corev1.KeyToPath{
				Key:  key,
				Path: path,
			},
		)
	}
}

// SecretMount returns a build option that adds a volume mount to a service or deployment.
func SecretMount(name, mountPath, secretName string, opts ...SecretMountOption) ObjectOption {
	return func(object interface{}) {
		vm := corev1.VolumeMount{
			Name:      name,
			ReadOnly:  true,
			MountPath: mountPath,
		}

		v := corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		}

		for _, opt := range opts {
			opt(&v, &vm)
		}

		var vols *[]corev1.Volume
		var volMounts *[]corev1.VolumeMount

		switch o := object.(type) {
		case *appsv1.Deployment:
			vols = &o.Spec.Template.Spec.Volumes
			volMounts = &firstContainer(o).VolumeMounts
		case *servingv1.Service:
			vols = &o.Spec.Template.Spec.Volumes
			volMounts = &firstContainer(o).VolumeMounts
		}

		*vols = append(*vols, v)
		*volMounts = append(*volMounts, vm)
	}
}
