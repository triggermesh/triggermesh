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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func TestNewServiceWithDefaultContainer(t *testing.T) {
	cpuRes, memRes := resource.MustParse("250m"), resource.MustParse("100Mi")

	v := corev1.Volume{
		Name: "some-volume",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: "some-secret",
				Items: []corev1.KeyToPath{{
					Key:  "someKey",
					Path: "someFile",
				}},
			},
		},
	}

	vm := corev1.VolumeMount{
		Name:      "some-volume",
		MountPath: "/myvol",
	}

	ksvc := NewKnService(tNs, tName,
		PodLabel("test.podlabel/2", "val2"),
		Port("health", 8081),
		Image(tImg),
		PodLabel("test.podlabel/1", "val1"),
		EnvVar("TEST_ENV1", "val1"),
		Port("h2c", 8080), // overrides previously defined port
		Label("test.label/1", "val1"),
		Probe("/health", "health"),             // port is ignored
		StartupProbe("/initialized", "health"), // port is ignored
		EnvVars(makeEnvVars(2, "MULTI_ENV", "val")...),
		EnvVar("TEST_ENV2", "val2"),
		Label("test.label/2", "val2"),
		ServiceAccount("god-mode"),
		Requests(&cpuRes, &memRes),
		Limits(&cpuRes, nil),
		Toleration(corev1.Toleration{Key: "taint", Operator: corev1.TolerationOpExists}),
		VisibilityClusterLocal,
		Volumes(v),
		VolumeMounts(vm),
	)

	expectKsvc := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
			Labels: map[string]string{
				"test.label/1":                      "val1",
				"test.label/2":                      "val2",
				"networking.knative.dev/visibility": "cluster-local",
			},
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"test.podlabel/1": "val1",
							"test.podlabel/2": "val2",
						},
					},
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							ServiceAccountName: "god-mode",
							Tolerations: []corev1.Toleration{{
								Key: "taint", Operator: "Exists",
							}},
							Containers: []corev1.Container{{
								Name:  defaultContainerName,
								Image: tImg,
								Ports: []corev1.ContainerPort{{
									Name:          "h2c",
									ContainerPort: 8080,
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
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/health",
										},
									},
								},
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/initialized",
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
								VolumeMounts: []corev1.VolumeMount{{
									Name:      "some-volume",
									MountPath: "/myvol",
								}},
							}},
							Volumes: []corev1.Volume{{
								Name: "some-volume",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: "some-secret",
										Items: []corev1.KeyToPath{{
											Key:  "someKey",
											Path: "someFile",
										}},
									},
								}},
							},
						},
					},
				},
			},
		},
	}

	if d := cmp.Diff(expectKsvc, ksvc); d != "" {
		t.Errorf("Unexpected diff: (-:expect, +:got) %s", d)
	}
}

func TestNewServiceWithCustomContainer(t *testing.T) {
	ksvc := NewKnService(tNs, tName,
		Container(&corev1.Container{Name: "foo"}),
	)

	expectKsvc := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name: "foo",
							}},
						},
					},
				},
			},
		},
	}

	if d := cmp.Diff(expectKsvc, ksvc); d != "" {
		t.Errorf("Unexpected diff: (-:expect, +:got) %s", d)
	}
}
