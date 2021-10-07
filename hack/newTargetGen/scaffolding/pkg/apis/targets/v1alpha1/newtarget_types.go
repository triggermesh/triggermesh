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

// $TARGETFULLCASE is the Schema for an Alibaba Object Storage Service Target.
type $TARGETFULLCASE struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   $TARGETFULLCASESpec   `json:"spec"`
	Status $TARGETFULLCASEStatus `json:"status,omitempty"`
}

// Check the interfaces $TARGETFULLCASE should be implementing.
var (
	_ runtime.Object            = (*$TARGETFULLCASE)(nil)
	_ kmeta.OwnerRefable        = (*$TARGETFULLCASE)(nil)
	_ targets.IntegrationTarget = (*$TARGETFULLCASE)(nil)
	_ targets.EventSource       = (*$TARGETFULLCASE)(nil)
	_ duckv1.KRShaped           = (*$TARGETFULLCASE)(nil)
)

// $TARGETFULLCASESpec holds the desired state of the $TARGETFULLCASE.
type $TARGETFULLCASESpec struct {

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// $TARGETFULLCASEStatus communicates the observed state of the $TARGETFULLCASE (from the controller).
type $TARGETFULLCASEStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`

	// Accepted/emitted CloudEvent attributes
	CloudEventStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// $TARGETFULLCASEList is a list of $TARGETFULLCASE resources
type $TARGETFULLCASEList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []$TARGETFULLCASE `json:"items"`
}
