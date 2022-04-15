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

// AlibabaOSSTarget is the Schema for an Alibaba Object Storage Service Target.
type AlibabaOSSTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlibabaOSSTargetSpec `json:"spec"`
	Status v1alpha1.Status      `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*AlibabaOSSTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*AlibabaOSSTarget)(nil)
	_ v1alpha1.EventReceiver       = (*AlibabaOSSTarget)(nil)
	_ v1alpha1.EventSource         = (*AlibabaOSSTarget)(nil)
)

// AlibabaOSSTargetSpec holds the desired state of the AlibabaOSSTarget.
type AlibabaOSSTargetSpec struct {
	// Alibaba SDK access key id as registered. For more information on how to
	// create an access key pair, please refer to
	// https://www.alibabacloud.com/help/doc-detail/53045.htm?spm=a2c63.p38356.879954.9.23bc7d91ARN6Hy#task968.
	AccessKeyID SecretValueFromSource `json:"accessKeyID"`

	// Alibaba SDK access key secret as registered.
	AccessKeySecret SecretValueFromSource `json:"accessKeySecret"`

	// The domain name used to access the OSS. For more information, please refer
	// to the region and endpoint guide at
	// https://www.alibabacloud.com/help/doc-detail/31837.htm?spm=a2c63.p38356.879954.8.23bc7d91ARN6Hy#concept-zt4-cvy-5db
	Endpoint string `json:"endpoint"`

	// The unique container to store objects in OSS.
	Bucket string `json:"bucket"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AlibabaOSSTargetList is a list of AlibabaOSSTarget resources
type AlibabaOSSTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AlibabaOSSTarget `json:"items"`
}
