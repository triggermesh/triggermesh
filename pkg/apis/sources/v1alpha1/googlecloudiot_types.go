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

// GoogleCloudIoTSource is the Schema for the event source.
type GoogleCloudIoTSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoogleCloudIoTSourceSpec   `json:"spec,omitempty"`
	Status GoogleCloudIoTSourceStatus `json:"status,omitempty"`
}

// Check the interfaces the event source should be implementing.
var (
	_ runtime.Object = (*GoogleCloudIoTSource)(nil)
	_ EventSource    = (*GoogleCloudIoTSource)(nil)
)

// GoogleCloudIoTSourceSpec defines the desired state of the event source.
type GoogleCloudIoTSourceSpec struct {
	duckv1.SourceSpec `json:",inline"`

	// Name of the Cloud IoT Registry to receive notifications from.
	Registry GCloudIoTResourceName `json:"registry"`

	// Settings related to the Pub/Sub resources associated with the repo events.
	PubSub GoogleCloudIoTSourcePubSubSpec `json:"pubsub"`

	// Service account key in JSON format.
	// https://cloud.google.com/iam/docs/creating-managing-service-account-keys
	ServiceAccountKey ValueFromField `json:"serviceAccountKey"`
}

// GoogleCloudIoTSourcePubSubSpec defines the attributes related to the
// configuration of Pub/Sub resources.
type GoogleCloudIoTSourcePubSubSpec struct {
	// Optional: no more than one of the following may be specified.

	// Full resource name of the Pub/Sub topic where change notifications
	// originating from the configured sink are sent to. If not supplied,
	// a topic is created on behalf of the user, in the GCP project
	// referenced by the Project attribute.
	//
	// The expected format is described at https://cloud.google.com/pubsub/docs/admin#resource_names:
	//   "projects/{project_name}/topics/{topic_name}"
	//
	// +optional
	Topic *GCloudResourceName `json:"topic,omitempty"`

	// Name of the GCP project where Pub/Sub resources associated with the
	// Cloud IoT are to be created.
	//
	// Mutually exclusive with Topic which, if supplied, already contains
	// the project name.
	//
	// +optional
	Project *string `json:"project,omitempty"`
}

// GoogleCloudIoTSourceStatus defines the observed state of the event source.
type GoogleCloudIoTSourceStatus struct {
	EventSourceStatus `json:",inline"`

	// Resource name of the target Pub/Sub topic.
	Topic *GCloudResourceName `json:"topic,omitempty"`

	// Resource name of the managed Pub/Sub subscription associated with
	// the managed topic.
	Subscription *GCloudResourceName `json:"subscription,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GoogleCloudIoTSourceList contains a list of event sources.
type GoogleCloudIoTSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GoogleCloudIoTSource `json:"items"`
}

// GCloudIoTResourceName represents a fully qualified IoT resource name,
// as described at
//  https://pkg.go.dev/google.golang.org/api/cloudiot/v1#DeviceRegistry.Name
//
// Examples of such resource names include:
//  - projects/{project_name}/locations/{location_name}/registries/{registry_name}
type GCloudIoTResourceName struct {
	Project    string
	Location   string
	Collection string
	Resource   string
}

var (
	_ fmt.Stringer     = (*GCloudIoTResourceName)(nil)
	_ json.Marshaler   = (*GCloudIoTResourceName)(nil)
	_ json.Unmarshaler = (*GCloudIoTResourceName)(nil)
)

const (
	GCloudIoTResourceNameFormat        = "projects/{project_name}/locations/{location_name}/{resource_type}/{resource_name}"
	GCloudIoTResourceNameSplitElements = 6
)

// UnmarshalJSON implements json.Unmarshaler
func (n *GCloudIoTResourceName) UnmarshalJSON(data []byte) error {
	var dataStr string
	if err := json.Unmarshal(data, &dataStr); err != nil {
		return err
	}

	sections := strings.Split(dataStr, "/")
	if len(sections) != GCloudIoTResourceNameSplitElements {
		return newParseGCloudIoTResourceNameError(dataStr)
	}

	const (
		projectIdx  = 1
		locationIdx = 3
		typeIdx     = 4
		resourceIdx = 5
	)

	project := sections[projectIdx]
	location := sections[locationIdx]
	typ := sections[typeIdx]
	resource := sections[resourceIdx]

	if project == "" || location == "" || typ == "" || resource == "" {
		return errGCloudIoTResourceNameEmptyAttrs
	}

	n.Project = project
	n.Location = location
	n.Collection = typ
	n.Resource = resource

	return nil
}

// MarshalJSON implements json.Marshaler
func (n GCloudIoTResourceName) MarshalJSON() ([]byte, error) {
	if n.Project == "" || n.Location == "" || n.Collection == "" || n.Resource == "" {
		return nil, errGCloudIoTResourceNameEmptyAttrs
	}

	var b bytes.Buffer

	b.WriteByte('"')
	b.WriteString("projects/")
	b.WriteString(n.Project)
	b.WriteByte('/')
	b.WriteString("locations/")
	b.WriteString(n.Location)
	b.WriteByte('/')
	b.WriteString(n.Collection)
	b.WriteByte('/')
	b.WriteString(n.Resource)
	b.WriteByte('"')

	return b.Bytes(), nil
}

// String implements the fmt.Stringer interface.
func (n *GCloudIoTResourceName) String() string {
	b, err := n.MarshalJSON()
	if err != nil {
		return ""
	}

	// skip checks on slice bound and leading/trailing quotes since we know
	// exactly what MarshalJSON returns
	return string(b[1 : len(b)-1])
}

// errGCloudIoTResourceNameEmptyAttrs indicates that a resource name string
// or object contains empty attributes.
var errGCloudIoTResourceNameEmptyAttrs = errors.New("resource name contains empty attributes")

// errParseGCloudIoTResourceName indicates that a resource ID string does
// not match the expected format.
type errParseGCloudIoTResourceName struct {
	gotInput string
}

func newParseGCloudIoTResourceNameError(got string) error {
	return &errParseGCloudIoTResourceName{
		gotInput: got,
	}
}

// Error implements the error interface.
func (e *errParseGCloudIoTResourceName) Error() string {
	return fmt.Sprintf("Resource name %q does not match expected format %q",
		e.gotInput, GCloudIoTResourceNameFormat)
}
