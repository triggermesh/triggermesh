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
	"testing"

	"github.com/google/go-cmp/cmp"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNewContainer(t *testing.T) {
	cpuRes, memRes := resource.MustParse("250m"), resource.MustParse("100Mi")

	vm := corev1.VolumeMount{
		Name:      "some-volume",
		MountPath: "/myvol",
	}

	cont := NewContainer(tName,
		Port("h2c", 8080),
		Image(tImg),
		EnvVar("TEST_ENV1", "val1"),
		Port("health", 8081),
		EnvVars(makeEnvVars(2, "MULTI_ENV", "val")...),
		EnvVar("TEST_ENV2", "val2"),
		EntrypointCommand("test", "--verbose"),
		Probe("/health", "health"),
		StartupProbe("/initialized", "health"),
		EnvVarFromSecret("TEST_ENV3", "test-secret", "someKey"),
		Requests(&cpuRes, &memRes),
		Limits(&cpuRes, nil),
		TerminationErrorToLogs,
		VolumeMounts(vm),
	)

	expectCont := &corev1.Container{
		Name:    tName,
		Image:   tImg,
		Command: []string{"test", "--verbose"},
		Ports: []corev1.ContainerPort{{
			Name:          "h2c",
			ContainerPort: 8080,
		}, {
			Name:          "health",
			ContainerPort: 8081,
		}},
		Env: []corev1.EnvVar{{
			Name:  "TEST_ENV1",
			Value: "val1",
		}, {
			Name:  "MULTI_ENV1",
			Value: "val1",
		}, {
			Name:  "MULTI_ENV2",
			Value: "val2",
		}, {
			Name:  "TEST_ENV2",
			Value: "val2",
		}, {
			Name: "TEST_ENV3",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "someKey",
				},
			},
		}},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health",
					Port: intstr.FromString("health"),
				},
			},
		},
		StartupProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/initialized",
					Port: intstr.FromString("health"),
				},
			},
			PeriodSeconds:    1,
			FailureThreshold: 60,
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewMilliQuantity(250, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(1024*1024*100, resource.BinarySI),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: *resource.NewMilliQuantity(250, resource.DecimalSI),
			},
		},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "some-volume",
				MountPath: "/myvol",
			},
		},
	}

	if d := cmp.Diff(expectCont, cont); d != "" {
		t.Errorf("Unexpected diff: (-:expect, +:got) %s", d)
	}
}
