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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSComprehendTarget is the Schema for an AWS Comprehend Target.
type AWSComprehendTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSComprehendTargetSpec `json:"spec"`
	Status v1alpha1.Status         `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*AWSComprehendTarget)(nil)
	_ v1alpha1.AdapterConfigurable    = (*AWSComprehendTarget)(nil)
	_ v1alpha1.EventSource            = (*AWSComprehendTarget)(nil)
	_ v1alpha1.ServiceAccountProvider = (*AWSComprehendTarget)(nil)
)

// AWSComprehendTargetSpec defines the desired state of the event target.
type AWSComprehendTargetSpec struct {
	// AWS-specific authentication methods.
	Auth v1alpha1.AWSAuth `json:"auth"`

	// Region to use for calling into Comprehend API.
	Region string `json:"region"`

	// EventOptions for targets.
	EventOptions *EventOptions `json:"eventOptions,omitempty"`

	// Language code to use to interact with Comprehend. The supported list can be found at: https://docs.aws.amazon.com/comprehend/latest/dg/supported-languages.html
	Language string `json:"language"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSComprehendTargetList is a list of event target instances.
type AWSComprehendTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AWSComprehendTarget `json:"items"`
}
