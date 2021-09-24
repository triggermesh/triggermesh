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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudPubSubSource is the Schema for the event source.
type GoogleCloudPubSubSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudPubSubSourceSpec   `json:"spec,omitempty"`
	Status GoogleCloudPubSubSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ runtime.Object = (*GoogleCloudPubSubSource)(nil)
	_ EventSource    = (*GoogleCloudPubSubSource)(nil)
)

// GoogleCloudPubSubSourceSpec defines the desired state of the event source.
type GoogleCloudPubSubSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Full resource name of the Pub/Sub topic to subscribe to, in the
	// format "projects/{project_name}/topics/{topic_name}".
	Topic GCloudPubSubResourceName `json:"topic"`

	// ID of the subscription to use to pull messages from the topic.
	//
	// If supplied, this subscription must 1) exist and 2) belong to the
	// provided topic. Otherwise, a pull subscription to that topic is
	// created on behalf of the user.
	//
	// +optional
	SubscriptionID *string `json:"subscriptionID,omitempty"`

	// Service account key in JSON format.
	// https://cloud.google.com/iam/docs/creating-managing-service-account-keys
	ServiceAccountKey ValueFromField `json:"serviceAccountKey"`
}

// GoogleCloudPubSubSourceStatus defines the observed state of the event source.
type GoogleCloudPubSubSourceStatus struct {
	EventSourceStatus `json:",inline"`
	Subscription      *GCloudPubSubResourceName `json:"subscription,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudPubSubSourceList contains a list of event sources.
type GoogleCloudPubSubSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GoogleCloudPubSubSource `json:"items"`
}

// GCloudPubSubResourceName represents a fully qualified Pub/Sub resource name,
// as described at
//  https://cloud.google.com/pubsub/docs/admin#resource_names
//
// Examples of such resource names include:
//  - projects/{project_name}/topics/{topic_name}
//  - projects/{project_name}/subscriptions/{subscription_name}
type GCloudPubSubResourceName struct {
	Project    string
	Collection string
	Resource   string
}

var (
	_ fmt.Stringer     = (*GCloudPubSubResourceName)(nil)
	_ json.Marshaler   = (*GCloudPubSubResourceName)(nil)
	_ json.Unmarshaler = (*GCloudPubSubResourceName)(nil)
)

const (
	gcloudPubSubResourceNameFormat        = "projects/{project_name}/{resource_type}/{resource_name}"
	gcloudPubSubResourceNameSplitElements = 4
)

// UnmarshalJSON implements json.Unmarshaler
func (n *GCloudPubSubResourceName) UnmarshalJSON(data []byte) error {
	var dataStr string
	if err := json.Unmarshal(data, &dataStr); err != nil {
		return err
	}

	sections := strings.Split(dataStr, "/")
	if len(sections) != gcloudPubSubResourceNameSplitElements {
		return newParseGCloudPubSubResourceNameError(dataStr)
	}

	const (
		projectIdx  = 1
		typeIdx     = 2
		resourceIdx = 3
	)

	project := sections[projectIdx]
	typ := sections[typeIdx]
	resource := sections[resourceIdx]

	if project == "" || typ == "" || resource == "" {
		return errGCloudPubSubResourceNameEmptyAttrs
	}

	n.Project = project
	n.Collection = typ
	n.Resource = resource

	return nil
}

// MarshalJSON implements json.Marshaler
func (n GCloudPubSubResourceName) MarshalJSON() ([]byte, error) {
	if n.Project == "" || n.Collection == "" || n.Resource == "" {
		return nil, errGCloudPubSubResourceNameEmptyAttrs
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

// String implements the fmt.Stringer interface.
func (n *GCloudPubSubResourceName) String() string {
	b, err := n.MarshalJSON()
	if err != nil {
		return ""
	}

	// skip checks on slice bound and leading/trailing quotes since we know
	// exactly what MarshalJSON returns
	return string(b[1 : len(b)-1])
}

// errGCloudPubSubResourceNameEmptyAttrs indicates that a resource name string
// or object contains empty attributes.
var errGCloudPubSubResourceNameEmptyAttrs = errors.New("resource name contains empty attributes")

// errParseGCloudPubSubResourceName indicates that a resource ID string does
// not match the expected format.
type errParseGCloudPubSubResourceName struct {
	gotInput string
}

func newParseGCloudPubSubResourceNameError(got string) error {
	return &errParseGCloudPubSubResourceName{
		gotInput: got,
	}
}

// Error implements the error interface.
func (e *errParseGCloudPubSubResourceName) Error() string {
	return fmt.Sprintf("Pub/Sub resource name %q does not match expected format %q",
		e.gotInput, gcloudPubSubResourceNameFormat)
}
