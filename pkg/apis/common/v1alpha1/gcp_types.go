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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

const annotationGcpSA = "iam.gke.io/gcp-service-account"

// GcpServiceAccountAnnotation returns a functional option that sets the GCP
// Service Account annotation on Kubernetes ServiceAccount.
func GcpServiceAccountAnnotation(gcpSA string) resource.ServiceAccountOption {
	return func(sa *corev1.ServiceAccount) {
		metav1.SetMetaDataAnnotation(&sa.ObjectMeta, annotationGcpSA, gcpSA)
	}
}

// K8sServiceAccountName returns a functional option that overwrites the
// Kubernetes Service Account name.
func K8sServiceAccountName(name string) resource.ServiceAccountOption {
	return func(sa *corev1.ServiceAccount) {
		sa.SetName(name)
	}
}
