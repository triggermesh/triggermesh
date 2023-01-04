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

package transformer

import (
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/storage"
)

// Transformer is an interface that contains common methods
// to work with JSON data.
type Transformer interface {
	New(key, value, separator string) Transformer
	Apply(eventID string, data []byte) ([]byte, error)
	SetStorage(*storage.Storage)
	InitStep() bool
}
