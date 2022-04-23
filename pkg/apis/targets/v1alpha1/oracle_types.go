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

// OracleTarget is the Schema for an Oracle Target.
type OracleTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OracleTargetSpec `json:"spec"`
	Status v1alpha1.Status  `json:"status,omitempty"`
}

// Check the interfaces the event target should be implementing.
var (
	_ v1alpha1.Reconcilable        = (*OracleTarget)(nil)
	_ v1alpha1.AdapterConfigurable = (*OracleTarget)(nil)
)

// OracleTargetSpec defines the desired state of the event target.
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

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`
}

// OracleFunctionSpecSpec defines the desired state of the event target.
type OracleFunctionSpecSpec struct {
	// Oracle Cloud ID of the function to invoke.
	Function string `json:"function,inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OracleTargetList is a list of event target instances.
type OracleTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []OracleTarget `json:"items"`
}
