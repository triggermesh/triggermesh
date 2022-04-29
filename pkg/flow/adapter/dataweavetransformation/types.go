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

package dataweavetransformation

// DataWeaveTransformationStructuredRequest contains an opinionated structure
// that informs both the InputData and Spell to transform.
type DataWeaveTransformationStructuredRequest struct {
	InputData         string `json:"input_data"`
	Spell             string `json:"spell,omitempty"`
	InputContentType  string `json:"input_content_type"`
	OutputContentType string `json:"output_content_type"`
}
