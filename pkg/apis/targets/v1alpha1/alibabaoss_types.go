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

package v1alpha1

import (
	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AlibabaOSSTarget is the Schema for an Alibaba Object Storage Service Target.
type AlibabaOSSTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlibabaOSSTargetSpec   `json:"spec"`
	Status AlibabaOSSTargetStatus `json:"status,omitempty"`
}

// Check the interfaces AlibabaOSSTarget should be implementing.
var (
	_ runtime.Object            = (*AlibabaOSSTarget)(nil)
	_ kmeta.OwnerRefable        = (*AlibabaOSSTarget)(nil)
	_ targets.IntegrationTarget = (*AlibabaOSSTarget)(nil)
	_ targets.EventSource       = (*AlibabaOSSTarget)(nil)
	_ duckv1.KRShaped           = (*AlibabaOSSTarget)(nil)
)

// AlibabaOSSTargetSpec holds the desired state of the AlibabaOSSTarget.
type AlibabaOSSTargetSpec struct {
	AccessKeyID SecretValueFromSource `json:"accessKeyID"`

	AccessKeySecret SecretValueFromSource `json:"accessKeySecret"`

	Endpoint string `json:"endpoint"`

	Bucket string `json:"bucket"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// AlibabaOSSTargetStatus communicates the observed state of the AlibabaOSSTarget (from the controller).
type AlibabaOSSTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`

	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AlibabaOSSTargetList is a list of AlibabaOSSTarget resources
type AlibabaOSSTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AlibabaOSSTarget `json:"items"`
}
