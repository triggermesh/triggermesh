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
	"path"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	network "knative.dev/networking/pkg"
	"knative.dev/pkg/kmeta"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	defaultContainerName = "user-container"
	mountPath            = "/opt"
)

// KnSvcOption sets Kn service options.
type KnSvcOption func(*servingv1.Service)

// NewKnService creates a Knative Service object.
func NewKnService(name, ns string, opts ...KnSvcOption) *servingv1.Service {
	d := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}

	for _, opt := range opts {
		opt(d)
	}

	// ensure the container name is not empty.
	containers := d.Spec.Template.Spec.Containers
	if len(containers) == 1 && containers[0].Name == "" {
		containers[0].Name = defaultContainerName
	}

	return d
}

// KnSvcImage sets a Container's image.
func KnSvcImage(img string) KnSvcOption {
	return func(svc *servingv1.Service) {
		image := &firstContainer(svc).Image
		*image = img
	}
}

// KnSvcEnvVars sets the value of multiple environment variables.
func KnSvcEnvVars(evs ...corev1.EnvVar) KnSvcOption {
	return func(svc *servingv1.Service) {
		svcEnvVars := envVarsFrom(svc)
		*svcEnvVars = append(*svcEnvVars, evs...)
	}
}

// KnSvcEnvVar sets the value of a Container's environment variable.
func KnSvcEnvVar(name, val string) KnSvcOption {
	return func(svc *servingv1.Service) {
		setEnvVar(envVarsFrom(svc), name, val, nil)
	}
}

// KnSvcAnnotation sets Kn service annotation.
func KnSvcAnnotation(key, value string) KnSvcOption {
	return func(svc *servingv1.Service) {
		if svc.Spec.Template.Annotations == nil {
			svc.Spec.Template.Annotations = make(map[string]string)
		}
		svc.Spec.Template.Annotations[key] = value
	}
}

// KnSvcOwner sets Kn service owner.
func KnSvcOwner(o kmeta.OwnerRefable) KnSvcOption {
	return func(svc *servingv1.Service) {
		svc.SetOwnerReferences([]metav1.OwnerReference{
			*kmeta.NewControllerRef(o),
		})
	}
}

// KnSvcLabel sets Kn service labels.
func KnSvcLabel(labels map[string]string) KnSvcOption {
	return func(svc *servingv1.Service) {
		if svc.Labels != nil {
			for k, v := range labels {
				svc.Labels[k] = v
			}
			return
		}
		svc.SetLabels(labels)
	}
}

// KnSvcMountCm sets Kn service volume mounts.
func KnSvcMountCm(cmSrc, fileDst string) KnSvcOption {
	return func(svc *servingv1.Service) {
		svc.Spec.ConfigurationSpec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "user-function",
				ReadOnly:  true,
				MountPath: path.Join(mountPath, fileDst),
				SubPath:   fileDst,
			},
		}
		svc.Spec.ConfigurationSpec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "user-function",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cmSrc,
						},
						Items: []corev1.KeyToPath{
							{
								Path: fileDst,
								Key:  defaultCmKey,
							},
						},
					},
				},
			},
		}
	}
}

// KnSvcEntrypoint sets Kn service entrypoint.
func KnSvcEntrypoint(command string) KnSvcOption {
	return func(svc *servingv1.Service) {
		svc.Spec.ConfigurationSpec.Template.Spec.Containers[0].Command = []string{command}
	}
}

// KnSvcVisibility sets Kn service visibility scope.
func KnSvcVisibility(public bool) KnSvcOption {
	return func(svc *servingv1.Service) {
		if public {
			return
		}
		if svc.Labels != nil {
			svc.Labels[network.VisibilityLabelKey] = serving.VisibilityClusterLocal
			return
		}
		labels := map[string]string{
			network.VisibilityLabelKey: serving.VisibilityClusterLocal,
		}
		svc.Labels = labels
	}
}

func envVarsFrom(svc *servingv1.Service) *[]corev1.EnvVar {
	return &firstContainer(svc).Env
}

func setEnvVar(envVars *[]corev1.EnvVar, name, value string, valueFrom *corev1.EnvVarSource) {
	*envVars = append(*envVars, corev1.EnvVar{
		Name:      name,
		Value:     value,
		ValueFrom: valueFrom,
	})
}

// firstContainer returns a PodSpecable's first Container definition.
// A new empty Container is injected if the PodSpecable does not contain any.
func firstContainer(svc *servingv1.Service) *corev1.Container {
	containers := &svc.Spec.Template.Spec.Containers
	if len(*containers) == 0 {
		*containers = make([]corev1.Container, 1)
	}
	return &(*containers)[0]
}
