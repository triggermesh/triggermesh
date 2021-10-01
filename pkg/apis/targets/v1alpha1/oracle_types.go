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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OracleTarget is the Schema for an Oracle Target.
type OracleTarget struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the OracleTarget (from the client).
	Spec OracleTargetSpec `json:"spec"`

	// Status communicates the observed state of the OracleTarget (from the controller).
	// +optional
	Status OracleTargetStatus `json:"status,omitempty"`
}

// Check the interfaces OracleTarget should be implementing.
var (
	_ runtime.Object     = (*OracleTarget)(nil)
	_ kmeta.OwnerRefable = (*OracleTarget)(nil)
	_ duckv1.KRShaped    = (*OracleTarget)(nil)
)

type OracleTargetSpec struct {
	// Oracle User API private key.
	OracleAPIPrivateKey SecretValueFromSource `json:"oracleApiPrivateKey"`

	// Oracle User API private key passphrase.
	OracleAPIPrivateKeyPassphrase SecretValueFromSource `json:"oracleApiPrivateKeyPassphrase"`

	// Oracle User API cert fingerprint.
	OracleAPIPrivateKeyFingerprint SecretValueFromSource `json:"oracleApiPrivateKeyFingerprint"`

	// Oracle Tenancy OCID.
	Tenancy string `json:"oracleTenancy"`

	// Oracle User OCID associated with the API key.
	User string `json:"oracleUser"`

	// Oracle Cloud Region.
	Region string `json:"oracleRegion"`

	OracleFunctionSpec *OracleFunctionSpecSpec `json:"function,omitempty"`
}

type OracleFunctionSpecSpec struct {
	// Oracle Cloud ID of the function to invoke.
	Function string `json:"function,inline"`
}

// OracleTargetStatus communicates the observed state of the OracleTarget (from the controller).
type OracleTargetStatus struct {
	// inherits duck/v1beta1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last
	//   processed by the controller.
	// * Conditions - the latest available observations of a resource's current
	//   state.
	duckv1.Status `json:",inline"`

	// AddressStatus fulfills the Addressable contract.
	duckv1.AddressStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OracleTargetList is a list of OracleTarget resources
type OracleTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []OracleTarget `json:"items"`
}
