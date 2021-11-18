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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	network "knative.dev/networking/pkg"
	"knative.dev/pkg/kmeta"
	serving "knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// KsvcOpts configures Knative service.
type KsvcOpts func(*servingv1.Service) *servingv1.Service

// MakeKService generates a Knative service.
func MakeKService(namespace, name, image string, opts ...KsvcOpts) *servingv1.Service {

	ksvc := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  AdapterComponent,
								Image: image,
							}},
						},
					},
				},
			},
		},
	}

	for _, f := range opts {
		ksvc = f(ksvc)
	}

	return ksvc
}

// KsvcOwner sets owner ref.
func KsvcOwner(owner kmeta.OwnerRefable) KsvcOpts {
	return func(ksvc *servingv1.Service) *servingv1.Service {
		ksvc.OwnerReferences = []metav1.OwnerReference{
			*kmeta.NewControllerRef(owner),
		}
		return ksvc
	}
}

// KsvcLabels sets labels.
func KsvcLabels(ls labels.Set) KsvcOpts {
	return func(ksvc *servingv1.Service) *servingv1.Service {
		ksvc.SetLabels(ls)
		return ksvc
	}
}

// KsvcLabelVisibilityClusterLocal sets label to avoid exposing the service externally.
func KsvcLabelVisibilityClusterLocal(ksvc *servingv1.Service) *servingv1.Service {
	ksvc.Labels[network.VisibilityLabelKey] = serving.VisibilityClusterLocal
	return ksvc
}

// KsvcServiceAccount sets the ServiceAccount.
func KsvcServiceAccount(serviceaccount string) KsvcOpts {
	return func(ksvc *servingv1.Service) *servingv1.Service {
		ksvc.Spec.Template.Spec.ServiceAccountName = serviceaccount
		return ksvc
	}
}

// KsvcPodLabels sets pod labels.
func KsvcPodLabels(ls labels.Set) KsvcOpts {
	return func(ksvc *servingv1.Service) *servingv1.Service {
		ksvc.Spec.Template.SetLabels(ls)
		return ksvc
	}
}

// KsvcPodEnvVars sets pod's first container env vars.
func KsvcPodEnvVars(env []corev1.EnvVar) KsvcOpts {
	return func(ksvc *servingv1.Service) *servingv1.Service {
		ksvc.Spec.Template.Spec.Containers[0].Env = env
		return ksvc
	}
}

// EnvVar sets the value of a Container's environment variable.
func EnvVar(name, val string) KsvcOpts {
	return func(object *servingv1.Service) *servingv1.Service {
		setEnvVar(envVarsFrom(object), name, val, nil)
		return object
	}
}

func envVarsFrom(object interface{}) (envVars *[]corev1.EnvVar) {
	switch o := object.(type) {
	case *corev1.Container:
		envVars = &o.Env
	case *appsv1.Deployment, *servingv1.Service:
		envVars = &firstContainer(o).Env
	}

	return
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
func firstContainer(object interface{}) *corev1.Container {
	var containers *[]corev1.Container

	switch o := object.(type) {
	case *appsv1.Deployment:
		containers = &o.Spec.Template.Spec.Containers
	case *servingv1.Service:
		containers = &o.Spec.Template.Spec.Containers
	}

	if len(*containers) == 0 {
		*containers = make([]corev1.Container, 1)
	}
	return &(*containers)[0]
}
