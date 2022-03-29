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

package googlecloudpubsubsource

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/kelseyhightower/envconfig"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

// GCloudResourceName is a fully qualified Google Cloud resource name which can
// be decoded by envconfig.
type GCloudResourceName v1alpha1.GCloudResourceName

var (
	_ fmt.Stringer      = (*GCloudResourceName)(nil)
	_ envconfig.Decoder = (*GCloudResourceName)(nil)
)

// String implements fmt.Stringer.
func (n *GCloudResourceName) String() string {
	return (*v1alpha1.GCloudResourceName)(n).String()
}

// Decode implements envconfig.Decoder.
func (n *GCloudResourceName) Decode(value string) error {
	resName := (*v1alpha1.GCloudResourceName)(n)
	return json.Unmarshal([]byte(strconv.Quote(value)), resName)
}
