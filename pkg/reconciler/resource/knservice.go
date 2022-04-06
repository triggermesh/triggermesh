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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	network "knative.dev/networking/pkg"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// NewKnService creates a Knative Service object.
func NewKnService(ns, name string, opts ...ObjectOption) *servingv1.Service {
	ks := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}

	for _, opt := range opts {
		opt(ks)
	}

	// ensure the container name is not empty
	containers := ks.Spec.Template.Spec.Containers
	if len(containers) == 1 && containers[0].Name == "" {
		containers[0].Name = defaultContainerName
	}

	return ks
}

// VisibilityClusterLocal makes the Knative Service only available on the
// cluster's local network.
func VisibilityClusterLocal(object interface{}) {
	ks := object.(*servingv1.Service)
	Label(network.VisibilityLabelKey, serving.VisibilityClusterLocal)(ks)
}

// VisibilityPublic makes the Knative Service available on the public internet.
func VisibilityPublic(object interface{}) {
	ks := object.(*servingv1.Service)
	delete(ks.Labels, network.VisibilityLabelKey)
}
