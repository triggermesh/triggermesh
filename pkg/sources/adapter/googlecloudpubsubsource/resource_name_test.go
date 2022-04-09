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

package googlecloudpubsubsource_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/triggermesh/triggermesh/pkg/sources/adapter/googlecloudpubsubsource"
)

func TestStringerGCloudResourceName(t *testing.T) {
	n := &GCloudResourceName{
		Project:    "project",
		Collection: "collection",
		Resource:   "resource",
	}

	const expect = "projects/project/collection/resource"

	assert.Equal(t, expect, n.String())
}

func TestDecodeGCloudResourceName(t *testing.T) {
	n := &GCloudResourceName{}

	err := n.Decode("invalid_resource_name")
	assert.Error(t, err)

	err = n.Decode("projects/project/collection/resource")
	require.NoError(t, err)

	assert.Equal(t, "project", n.Project)
	assert.Equal(t, "collection", n.Collection)
	assert.Equal(t, "resource", n.Resource)
}
