/*
Copyright (c) 2020-2021 TriggerMesh Inc.

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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNewDeploymentWithDefaultContainer(t *testing.T) {
	depl := NewDeployment(tNs, tName,
		PodLabel("test.podlabel/2", "val2"),
		Selector("test.selector/1", "val1"),
		Port("h2c", 8080),
		Image(tImg),
		PodLabel("test.podlabel/1", "val1"),
		EnvVar("TEST_ENV1", "val1"),
		Selector("test.selector/2", "val2"),
		Port("health", 8081),
		Label("test.label/1", "val1"),
		Probe("/health", "health"),
		StartupProbe("/initialized", "health"),
		EnvVars(makeEnvVars(2, "MULTI_ENV", "val")...),
		EnvVar("TEST_ENV2", "val2"),
		Label("test.label/2", "val2"),
		ServiceAccount("god-mode"),
		Requests(resource.MustParse("250m"), resource.MustParse("100Mi")),
		Limits(resource.MustParse("250m"), resource.MustParse("100Mi")),
		TerminationErrorToLogs,
	)

	expectDepl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
			Labels: map[string]string{
				"test.label/1": "val1",
				"test.label/2": "val2",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"test.selector/1": "val1",
					"test.selector/2": "val2",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"test.selector/1": "val1",
						"test.selector/2": "val2",
						"test.podlabel/1": "val1",
						"test.podlabel/2": "val2",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "god-mode",
					Containers: []corev1.Container{{
						Name:  defaultContainerName,
						Image: tImg,
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
						}},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/health",
									Port: intstr.FromString("health"),
								},
							},
							InitialDelaySeconds: 2,
						},
						StartupProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/initialized",
									Port: intstr.FromString("health"),
								},
							},
							PeriodSeconds: 1,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    *resource.NewMilliQuantity(250, resource.DecimalSI),
								corev1.ResourceMemory: *resource.NewQuantity(1024*1024*100, resource.BinarySI),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    *resource.NewMilliQuantity(250, resource.DecimalSI),
								corev1.ResourceMemory: *resource.NewQuantity(1024*1024*100, resource.BinarySI),
							},
						},
						TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
					}},
				},
			},
		},
	}

	if d := cmp.Diff(expectDepl, depl); d != "" {
		t.Errorf("Unexpected diff: (-:expect, +:got) %s", d)
	}
}

func TestNewDeploymentWithCustomContainer(t *testing.T) {
	depl := NewDeployment(tNs, tName,
		Container(&corev1.Container{Name: "foo"}),
	)

	expectDepl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "foo",
					}},
				},
			},
		},
	}

	if d := cmp.Diff(expectDepl, depl); d != "" {
		t.Errorf("Unexpected diff: (-:expect, +:got) %s", d)
	}
}
