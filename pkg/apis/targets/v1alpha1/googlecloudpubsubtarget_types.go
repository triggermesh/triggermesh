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
	"bytes"
	"errors"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudPubSubTarget is the Schema the event target.
type GoogleCloudPubSubTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudPubSubTargetSpec   `json:"spec"`
	Status GoogleCloudPubSubTargetStatus `json:"status,omitempty"`
}

// Check the interfaces GoogleCloudPubSubTarget should be implementing.
var (
	_ runtime.Object            = (*GoogleCloudPubSubTarget)(nil)
	_ kmeta.OwnerRefable        = (*GoogleCloudPubSubTarget)(nil)
	_ targets.IntegrationTarget = (*GoogleCloudPubSubTarget)(nil)
	_ targets.EventSource       = (*GoogleCloudPubSubTarget)(nil)
	_ duckv1.KRShaped           = (*GoogleCloudPubSubTarget)(nil)
)

// GCloudResourceName represents a fully qualified resource name,
// as described at
//  https://cloud.google.com/apis/design/resource_names
//
// Examples of such resource names include:
//  - projects/{project_name}/topics/{topic_name}
//  - projects/{project_name}/repos/{repo_name}
//  - projects/{project_name}/subscriptions/{subscription_name}
type GCloudResourceName struct {
	Project    string
	Collection string
	Resource   string
}

// String implements the fmt.Stringer interface.
func (n *GCloudResourceName) String() string {
	b, err := n.MarshalJSON()
	if err != nil {
		return ""
	}

	// skip checks on slice bound and leading/trailing quotes since we know
	// exactly what MarshalJSON returns
	return string(b[1 : len(b)-1])
}

// errGCloudResourceNameEmptyAttrs indicates that a resource name string
// or object contains empty attributes.
var errGCloudResourceNameEmptyAttrs = errors.New("resource name contains empty attributes")

// MarshalJSON implements json.Marshaler
func (n GCloudResourceName) MarshalJSON() ([]byte, error) {
	if n.Project == "" || n.Collection == "" || n.Resource == "" {
		return nil, errGCloudResourceNameEmptyAttrs
	}

	var b bytes.Buffer

	b.WriteByte('"')
	b.WriteString("projects/")
	b.WriteString(n.Project)
	b.WriteByte('/')
	b.WriteString(n.Collection)
	b.WriteByte('/')
	b.WriteString(n.Resource)
	b.WriteByte('"')

	return b.Bytes(), nil
}

// GoogleCloudPubSubTargetSpec holds the desired state of the event target.
type GoogleCloudPubSubTargetSpec struct {

	// Full resource name of the Pub/Sub topic to subscribe to, in the
	// format "projects/{project_name}/topics/{topic_name}".
	Topic GCloudResourceName `json:"topic"`

	// Service account key in JSON format.
	// https://cloud.google.com/iam/docs/creating-managing-service-account-keys
	ServiceAccountKey v1alpha1.ValueFromField `json:"serviceAccountKey"`

	// Adapter spec overrides parameters.
	// +optional
	AdapterOverrides *v1alpha1.AdapterOverrides `json:"adapterOverrides,omitempty"`

	// EventOptions for targets
	EventOptions *EventOptions `json:"eventOptions,omitempty"`
}

// GoogleCloudPubSubTargetStatus communicates the observed state of the event target. (from the controller).
type GoogleCloudPubSubTargetStatus struct {
	duckv1.Status        `json:",inline"`
	duckv1.AddressStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudPubSubTargetList is a list of event target instances.
type GoogleCloudPubSubTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GoogleCloudPubSubTarget `json:"items"`
}
