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
	"k8s.io/apimachinery/pkg/runtime"
)

// ValueFromField is a struct field that can have its value either defined
// explicitly or sourced from another entity.
//
// +k8s:deepcopy-gen=true
type ValueFromField struct {
	// Optional: no more than one of the following may be specified.

	// Field value.
	// +optional
	Value string `json:"value,omitempty"`
	// Field value from a Kubernetes Secret.
	// +optional
	ValueFromSecret *corev1.SecretKeySelector `json:"valueFromSecret,omitempty"`
}

// AdapterOverrides are applied on top of the default adapter parameters.
//
// +k8s:deepcopy-gen=true
type AdapterOverrides struct {
	// Public value indicates if the adapter backed by a Kn Service should have
	// its network visibility scope set to public. Default scope is cluster-local.
	Public *bool `json:"public,omitempty"`
	// Resources limits and requirements applied on adapter container.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Pod tolerations.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// NodeSelector to control which nodes the pod can be scheduled on.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Pod affinity.
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Environment variables applied on adapter container.
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Labels applied on adapter container.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations applied on adapter container.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// GroupObject holds the API group object types.
//
// +k8s:deepcopy-gen=false
type GroupObject struct {
	Single, List runtime.Object
}
