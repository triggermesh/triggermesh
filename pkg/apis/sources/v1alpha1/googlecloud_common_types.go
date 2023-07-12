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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// GCloudResourceName represents a fully qualified resource name,
// as described at
//
//	https://cloud.google.com/apis/design/resource_names
//
// Examples of such resource names include:
//   - projects/{project_name}/topics/{topic_name}
//   - projects/{project_name}/repos/{repo_name}
//   - projects/{project_name}/subscriptions/{subscription_name}
type GCloudResourceName struct {
	Project    string
	Collection string
	Resource   string
}

var (
	_ fmt.Stringer     = (*GCloudResourceName)(nil)
	_ json.Marshaler   = (*GCloudResourceName)(nil)
	_ json.Unmarshaler = (*GCloudResourceName)(nil)
)

const (
	gcloudResourceNameFormat        = "projects/{project_name}/{resource_type}/{resource_name}"
	gcloudResourceNameSplitElements = 4
)

// UnmarshalJSON implements json.Unmarshaler
func (n *GCloudResourceName) UnmarshalJSON(data []byte) error {
	var dataStr string
	if err := json.Unmarshal(data, &dataStr); err != nil {
		return err
	}

	sections := strings.Split(dataStr, "/")
	if len(sections) != gcloudResourceNameSplitElements {
		return newParseGCloudResourceNameError(dataStr)
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
		return errGCloudResourceNameEmptyAttrs
	}

	n.Project = project
	n.Collection = typ
	n.Resource = resource

	return nil
}

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

// errParseGCloudResourceName indicates that a resource ID string does
// not match the expected format.
type errParseGCloudResourceName struct {
	gotInput string
}

func newParseGCloudResourceNameError(got string) error {
	return &errParseGCloudResourceName{
		gotInput: got,
	}
}

// Error implements the error interface.
func (e *errParseGCloudResourceName) Error() string {
	return fmt.Sprintf("Resource name %q does not match expected format %q",
		e.gotInput, gcloudResourceNameFormat)
}

// GoogleCloudSourcePubSubSpec defines the attributes related to the
// configuration of Pub/Sub resources.
type GoogleCloudSourcePubSubSpec struct {
	// Optional: no more than one of the following may be specified.

	// Full resource name of the Pub/Sub topic where messages/notifications
	// originating from the configured Google Cloud resource are sent to,
	// before being retrieved by this event source. If not supplied, a
	// topic is created on behalf of the user, in the GCP project
	// referenced by the Project attribute.
	//
	// The expected format is described at https://cloud.google.com/pubsub/docs/admin#resource_names:
	//   "projects/{project_name}/topics/{topic_name}"
	//
	// +optional
	Topic *GCloudResourceName `json:"topic,omitempty"`

	// Name of the GCP project where Pub/Sub resources associated with the
	// configured Google Cloud resource are to be created.
	//
	// Mutually exclusive with Topic which, if supplied, already contains
	// the project name.
	//
	// +optional
	Project *string `json:"project,omitempty"`
}
