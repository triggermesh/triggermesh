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

package apps

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"

	"knative.dev/eventing/pkg/apis/duck"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateSimpleApplication is a helper which creates a simple Deployment
// exposed by a matching Service. `internalPort` is the TCP port number of the
// container managed by the Deployment. `exposedPort` is the TCP port number
// exposed by the Service.
func CreateSimpleApplication(c clientset.Interface, namespace string,
	name, image string, internalPort, exposedPort uint16,
	deplOpts ...DeploymentOption) (*appsv1.Deployment, *corev1.Service) {

	svc := newSimpleService(namespace, name, exposedPort, internalPort)

	svc, err := c.CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create Service: %s", err)
	}

	depl := newSimpleDeployment(namespace, name, image, internalPort)

	for _, o := range deplOpts {
		o(depl)
	}

	depl, err = c.AppsV1().Deployments(namespace).Create(context.Background(), depl, metav1.CreateOptions{})
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create Deployment: %s", err)
	}

	WaitUntilAvailable(c, depl)

	return depl, svc
}

// DeploymentOption is a functional option for building a Deployment object.
type DeploymentOption func(*appsv1.Deployment)

// WithStartupProbe sets the HTTP startup probe of a Deployment's first
// container, targetting this container's first TCP port.
func WithStartupProbe(path string) DeploymentOption {
	return func(d *appsv1.Deployment) {
		sp := &d.Spec.Template.Spec.Containers[0].StartupProbe

		port := int(d.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)

		*sp = &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: path,
					Port: intstr.FromInt(port),
				},
			},
			PeriodSeconds: 1,
		}
	}
}

// WaitUntilAvailable waits until the given Deployment becomes available.
func WaitUntilAvailable(c clientset.Interface, d *appsv1.Deployment) {
	fieldSelector := fields.OneTermEqualSelector("metadata.name", d.Name).String()

	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return c.AppsV1().Deployments(d.Namespace).List(context.Background(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return c.AppsV1().Deployments(d.Namespace).Watch(context.Background(), options)
		},
	}

	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}

	// checks whether the Deployment referenced in the given watch.Event is available.
	var isDeploymentAvailable watchtools.ConditionFunc = func(e watch.Event) (bool, error) {
		if e.Type == watch.Deleted {
			return false, apierrors.NewNotFound(gr, d.Name)
		}

		if d, ok := e.Object.(*appsv1.Deployment); ok {
			return duck.DeploymentIsAvailable(&d.Status, false), nil
		}

		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	_, err := watchtools.UntilWithSync(ctx, lw, &appsv1.Deployment{}, nil, isDeploymentAvailable)
	if err != nil {
		framework.FailfWithOffset(2, "Error waiting for %s %q to become available: %s", gr, d.Name, err)
	}
}

// newSimpleDeployment returns a Deployment object with a single container and
// default settings.
func newSimpleDeployment(namespace, name, image string, containerPort uint16) *appsv1.Deployment {
	const containerName = "app"

	lbls := labels.Set{
		labelAppName:   name,
		labelManagedBy: labelManagedByVal,
	}

	metadata := metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		Labels:    lbls,
	}

	return &appsv1.Deployment{
		ObjectMeta: metadata,
		Spec: appsv1.DeploymentSpec{
			Selector: metav1.SetAsLabelSelector(lbls),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lbls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  containerName,
						Image: image,
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(containerPort),
						}},
					}},
				},
			},
		},
	}
}

// newSimpleService returns a Service object with default settings.
func newSimpleService(namespace, name string, port, targetPort uint16) *corev1.Service {
	lbls := labels.Set{
		labelAppName:   name,
		labelManagedBy: labelManagedByVal,
	}

	metadata := metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		Labels:    lbls,
	}

	return &corev1.Service{
		ObjectMeta: metadata,
		Spec: corev1.ServiceSpec{
			Selector: lbls,
			Ports: []corev1.ServicePort{{
				Port:       int32(port),
				TargetPort: intstr.FromInt(int(targetPort)),
			}},
		},
	}
}
