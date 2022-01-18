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

package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	network "knative.dev/networking/pkg"
	"knative.dev/pkg/kmeta"
	serving "knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const defaultContainerName = "transformer"

// Option sets kn service options
type Option func(*servingv1.Service)

// NewKnService creates a Knative Service object.
func NewKnService(ns, name string, opts ...Option) *servingv1.Service {
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
func Image(img string) Option {
	return func(svc *servingv1.Service) {
		image := &firstContainer(svc).Image
		*image = img
	}
}

// EnvVars sets the value of multiple environment variables.
func EnvVars(evs ...corev1.EnvVar) Option {
	return func(svc *servingv1.Service) {
		svcEnvVars := envVarsFrom(svc)
		*svcEnvVars = append(*svcEnvVars, evs...)
	}
}

// EnvVar sets the value of a Container's environment variable.
func EnvVar(name, val string) Option {
	return func(svc *servingv1.Service) {
		setEnvVar(envVarsFrom(svc), name, val, nil)
	}
}

// Owner sets service owner.
func Owner(o kmeta.OwnerRefable) Option {
	return func(svc *servingv1.Service) {
		svc.SetOwnerReferences([]metav1.OwnerReference{
			*kmeta.NewControllerRef(o),
		})
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

// KsvcLabelVisibilityClusterLocal sets label to avoid exposing the service externally.
func KsvcLabelVisibilityClusterLocal() Option {
	return func(svc *servingv1.Service) {
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
