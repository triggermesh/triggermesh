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
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	network "knative.dev/networking/pkg"
	"knative.dev/pkg/kmeta"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	defaultContainerName = "user-container"
	MountPath            = "/opt"
)

// Option sets kn service options
type knSvcOption func(*servingv1.Service)

// NewKnService creates a Knative Service object.
func NewKnService(name, ns string, opts ...knSvcOption) *servingv1.Service {
	d := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}

	for _, opt := range opts {
		opt(d)
	}

	// ensure the container name is not empty
	containers := d.Spec.Template.Spec.Containers
	if len(containers) == 1 && containers[0].Name == "" {
		containers[0].Name = defaultContainerName
	}

	return d
}

// Image sets a Container's image.
func KnSvcImage(img string) knSvcOption {
	return func(svc *servingv1.Service) {
		image := &firstContainer(svc).Image
		*image = img
	}
}

// EnvVars sets the value of multiple environment variables.
func KnSvcEnvVars(evs ...corev1.EnvVar) knSvcOption {
	return func(svc *servingv1.Service) {
		svcEnvVars := envVarsFrom(svc)
		*svcEnvVars = append(*svcEnvVars, evs...)
	}
}

// EnvVar sets the value of a Container's environment variable.
func KnSvcEnvVar(name, val string) knSvcOption {
	return func(svc *servingv1.Service) {
		setEnvVar(envVarsFrom(svc), name, val, nil)
	}
}

func KnSvcAnnotation(key, value string) knSvcOption {
	return func(svc *servingv1.Service) {
		if svc.Spec.Template.Annotations == nil {
			svc.Spec.Template.Annotations = make(map[string]string)
		}
		svc.Spec.Template.Annotations[key] = value
	}
}

func KnSvcOwner(o kmeta.OwnerRefable) knSvcOption {
	return func(svc *servingv1.Service) {
		svc.SetOwnerReferences([]metav1.OwnerReference{
			*kmeta.NewControllerRef(o),
		})
	}
}

func KnSvcLabel(labels map[string]string) knSvcOption {
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

func KnSvcMountCm(cmSrc, fileDst string) knSvcOption {
	return func(svc *servingv1.Service) {
		svc.Spec.ConfigurationSpec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "user-function",
				ReadOnly:  true,
				MountPath: path.Join(MountPath, fileDst),
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
								Key:  DefaultCmKey,
							},
						},
					},
				},
			},
		}
	}
}

func KnSvcEntrypoint(command string) knSvcOption {
	return func(svc *servingv1.Service) {
		svc.Spec.ConfigurationSpec.Template.Spec.Containers[0].Command = []string{command}
	}
}

func KnSvcVisibility(public bool) knSvcOption {
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

func KnSvcEnvFromMap(prefix string, vars map[string]string) knSvcOption {
	return func(svc *servingv1.Service) {
		svcEnvVars := envVarsFrom(svc)
		for k, v := range vars {
			*svcEnvVars = append(*svcEnvVars, corev1.EnvVar{
				Name:  strings.ToUpper(prefix + k),
				Value: v,
			})
		}
	}
}
