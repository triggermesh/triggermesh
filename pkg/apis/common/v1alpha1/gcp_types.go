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

// GoogleCloudAuth contains authentication related attributes.
//
// +k8s:deepcopy-gen=true
type GoogleCloudAuth struct {
	// Service account key in JSON format.
	// https://cloud.google.com/iam/docs/creating-managing-service-account-keys
	ServiceAccountKey *ValueFromField `json:"serviceAccountKey,omitempty"`

	// GCP Service account for Workload Identity.
	// https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
	GCPServiceAccount *string `json:"gcpServiceAccount,omitempty"`

	// Name of the kubernetes service account bound to the gcpServiceAccount to act as an IAM service account.
	KubernetesServiceAccount *string `json:"kubernetesServiceAccount,omitempty"`
}

// gcpSaAnnotation returns a functional option that sets the GCP
// Service Account annotation on Kubernetes ServiceAccount.
func gcpSaAnnotation(gcpSA string) resource.ServiceAccountOption {
	return func(sa *corev1.ServiceAccount) {
		metav1.SetMetaDataAnnotation(&sa.ObjectMeta, annotationGcpSA, gcpSA)
	}
}

// WantsOwnServiceAccount indicates wether the object requires its own SA.
func (a *GoogleCloudAuth) WantsOwnServiceAccount() bool {
	return a.GCPServiceAccount != nil || a.KubernetesServiceAccount != nil
}

// ServiceAccountOptions is the set of mutations applied on the service account.
func (a *GoogleCloudAuth) ServiceAccountOptions() []resource.ServiceAccountOption {
	var saOpts []resource.ServiceAccountOption
	if a.GCPServiceAccount != nil {
		saOpts = append(saOpts, gcpSaAnnotation(*a.GCPServiceAccount))
	}
	if a.KubernetesServiceAccount != nil {
		saOpts = append(saOpts, saName(*a.KubernetesServiceAccount))
	}
	return saOpts
}
