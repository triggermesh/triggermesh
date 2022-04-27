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
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis"
	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSCognitoUserPoolSource is the Schema for the event source.
type AWSCognitoUserPoolSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSCognitoUserPoolSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status              `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable           = (*AWSCognitoUserPoolSource)(nil)
	_ v1alpha1.AdapterConfigurable    = (*AWSCognitoUserPoolSource)(nil)
	_ v1alpha1.EventSource            = (*AWSCognitoUserPoolSource)(nil)
	_ v1alpha1.EventSender            = (*AWSCognitoUserPoolSource)(nil)
	_ v1alpha1.ServiceAccountProvider = (*AWSCognitoUserPoolSource)(nil)
)

// AWSCognitoUserPoolSourceSpec defines the desired state of the event source.
type AWSCognitoUserPoolSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// User Pool ARN
	// https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazoncognitouserpools.html#amazoncognitouserpools-resources-for-iam-policies
	ARN apis.ARN `json:"arn"`

	// Authentication method to interact with the Amazon Cognito API.
	Auth AWSAuth `json:"auth"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSCognitoUserPoolSourceList contains a list of event sources.
type AWSCognitoUserPoolSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSCognitoUserPoolSource `json:"items"`
}
