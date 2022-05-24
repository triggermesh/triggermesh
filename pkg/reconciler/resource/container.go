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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const defaultContainerName = "adapter"

// NewContainer creates a Container object.
func NewContainer(name string, opts ...ObjectOption) *corev1.Container {
	c := &corev1.Container{
		Name: name,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Image sets a Container's image.
func Image(img string) ObjectOption {
	return func(object interface{}) {
		var image *string

		switch o := object.(type) {
		case *corev1.Container:
			image = &o.Image
		case *appsv1.Deployment, *servingv1.Service:
			image = &firstContainer(o).Image
		}

		*image = img
	}
}

// Port adds a port to a Container.
func Port(name string, port int32) ObjectOption {
	return func(object interface{}) {
		var ports *[]corev1.ContainerPort

		switch o := object.(type) {
		case *corev1.Container:
			ports = &o.Ports
		case *appsv1.Deployment, *servingv1.Service:
			ports = &firstContainer(o).Ports
		}

		switch object.(type) {
		case *corev1.Container, *appsv1.Deployment:
			*ports = append(*ports, corev1.ContainerPort{
				Name:          name,
				ContainerPort: port,
			})

		case *servingv1.Service:
			// Knative Services can only define 1 port
			*ports = []corev1.ContainerPort{{
				Name:          name,
				ContainerPort: port,
			}}
		}
	}
}

// EnvVar sets the value of a Container's environment variable.
func EnvVar(name, val string) ObjectOption {
	return func(object interface{}) {
		setEnvVar(envVarsFrom(object), name, val, nil)
	}
}

// EnvVars sets the value of multiple environment variables.
func EnvVars(evs ...corev1.EnvVar) ObjectOption {
	return func(object interface{}) {
		objEnvVars := envVarsFrom(object)
		*objEnvVars = append(*objEnvVars, evs...)
	}
}

// EnvVarFromSecret sets the value of a Container's environment variable to a
// reference to a Kubernetes Secret.
func EnvVarFromSecret(name, secretName, secretKey string) ObjectOption {
	return func(object interface{}) {
		valueFrom := &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: secretKey,
			},
		}

		setEnvVar(envVarsFrom(object), name, "", valueFrom)
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

// Probe sets the HTTP readiness probe of a Container or PodSpecable's first container.
func Probe(path, port string) ObjectOption {
	return func(object interface{}) {
		var rp **corev1.Probe
		var intstrPort intstr.IntOrString

		switch o := object.(type) {
		case *corev1.Container:
			rp = &o.ReadinessProbe
		case *appsv1.Deployment, *servingv1.Service:
			rp = &firstContainer(o).ReadinessProbe
		}

		switch object.(type) {
		case *corev1.Container, *appsv1.Deployment:
			intstrPort = intstr.FromString(port)
		case *servingv1.Service:
			// setting port explicitly is invalid in a Knative Service
		}

		*rp = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: path,
					Port: intstrPort,
				},
			},
		}
	}
}

// StartupProbe sets the HTTP startup probe of a Container or PodSpecable's first container.
func StartupProbe(path, port string) ObjectOption {
	return func(object interface{}) {
		var sp **corev1.Probe
		var intstrPort intstr.IntOrString

		switch o := object.(type) {
		case *corev1.Container:
			sp = &o.StartupProbe
		case *appsv1.Deployment, *servingv1.Service:
			sp = &firstContainer(o).StartupProbe
		}

		switch object.(type) {
		case *corev1.Container, *appsv1.Deployment:
			intstrPort = intstr.FromString(port)
		case *servingv1.Service:
			// setting port explicitly is invalid in a Knative Service
		}

		*sp = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: path,
					Port: intstrPort,
				},
			},
			PeriodSeconds:    1,
			FailureThreshold: 60,
		}
	}
}

// Requests sets the CPU and memory requests of a Container or PodSpecable's first container.
func Requests(cpu, mem *resource.Quantity) ObjectOption {
	return func(object interface{}) {
		setResources(&resourcesFrom(object).Requests, cpu, mem)
	}
}

// Limits sets the CPU and memory limits of a Container or PodSpecable's first container.
func Limits(cpu, mem *resource.Quantity) ObjectOption {
	return func(object interface{}) {
		setResources(&resourcesFrom(object).Limits, cpu, mem)
	}
}

func resourcesFrom(object interface{}) (resources *corev1.ResourceRequirements) {
	switch o := object.(type) {
	case *corev1.Container:
		resources = &o.Resources
	case *appsv1.Deployment, *servingv1.Service:
		resources = &firstContainer(o).Resources
	}

	return
}

func setResources(res *corev1.ResourceList, cpu, mem *resource.Quantity) {
	if *res == nil {
		*res = make(corev1.ResourceList, 2)
	}

	if cpu != nil {
		(*res)[corev1.ResourceCPU] = *cpu
	}
	if mem != nil {
		(*res)[corev1.ResourceMemory] = *mem
	}
}

// EntrypointCommand overrides the entrypoint command of a Container or
// PodSpecable's first container.
func EntrypointCommand(cmdAndArgs ...string) ObjectOption {
	return func(object interface{}) {
		var cmd *[]string

		switch o := object.(type) {
		case *corev1.Container:
			cmd = &o.Command
		case *appsv1.Deployment, *servingv1.Service:
			cmd = &firstContainer(o).Command
		}

		*cmd = cmdAndArgs
	}
}

// TerminationErrorToLogs sets the TerminationMessagePolicy of a container to
// FallbackToLogsOnError.
func TerminationErrorToLogs(object interface{}) {
	var tmp *corev1.TerminationMessagePolicy

	switch o := object.(type) {
	case *corev1.Container:
		tmp = &o.TerminationMessagePolicy
	case *appsv1.Deployment:
		tmp = &firstContainer(o).TerminationMessagePolicy
	}

	*tmp = corev1.TerminationMessageFallbackToLogsOnError
}

// VolumeMounts attaches VolumeMounts to a Container.
func VolumeMounts(vms ...corev1.VolumeMount) ObjectOption {
	return func(object interface{}) {
		var volMounts *[]corev1.VolumeMount

		switch o := object.(type) {
		case *corev1.Container:
			volMounts = &o.VolumeMounts
		case *appsv1.Deployment, *servingv1.Service:
			volMounts = &firstContainer(o).VolumeMounts
		}

		*volMounts = append(*volMounts, vms...)
	}
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
