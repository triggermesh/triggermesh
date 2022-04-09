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

	"github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

/* Provide common structs that are used by the targets such as secret definitions */

// ValueFromField is a struct field that can have its value either defined
// explicitly or sourced from another entity.
type ValueFromField struct {
	// Optional: no more than one of the following may be specified.

	// Field value.
	// +optional
	Value string `json:"value,omitempty"`
	// Field value from a Kubernetes Secret.
	// +optional
	ValueFromSecret *corev1.SecretKeySelector `json:"valueFromSecret,omitempty"`
	// Field value from a Kubernetes ConfigMap.
	// +optional
	ValueFromConfigMap *corev1.ConfigMapKeySelector `json:"valueFromConfigMap,omitempty"`
}

// EventOptions modifies CloudEvents management at Targets.
type EventOptions struct {
	// PayloadPolicy indicates if replies from the target should include
	// a payload if available. Possible values are:
	//
	// - always: will return a with the reply payload if avaliable.
	// - errors: will only reply with payload in case of an error.
	// - never: will not reply with payload.
	//
	// +optional
	PayloadPolicy *cloudevents.PayloadPolicy `json:"payloadPolicy,omitempty"`
}
