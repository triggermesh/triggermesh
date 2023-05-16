/*
Copyright 2023 TriggerMesh Inc.

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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MongoDBSource is the Schema for the event source.
type MongoDBSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoDBSourceSpec `json:"spec,omitempty"`
	Status v1alpha1.Status   `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ v1alpha1.Reconcilable = (*MongoDBSource)(nil)
	_ v1alpha1.EventSender  = (*MongoDBSource)(nil)
	_ v1alpha1.EventSource  = (*MongoDBSource)(nil)
)

// MongoDBSourceSpec defines the desired state of the event source.
type MongoDBSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// ConnectionString holds the connection string to the MongoDB server.
	ConnectionString string `json:"connectionString"`

	// Database holds the name of the MongoDB database.
	Database string `json:"database"`

	// Collection holds the name of the MongoDB collection.
	Collection string `json:"collection"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MongoDBSourceList contains a list of event sources.
type MongoDBSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoDBSource `json:"items"`
}
