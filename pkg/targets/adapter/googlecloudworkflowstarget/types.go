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

package googlecloudworkflowstarget

type CreateWorkflowEvent struct {
	Parent     string `json:"parent"`
	WorkflowID string `json:"workflow_id"`
	Workflow   struct {
		Name        string             `json:"name"`
		State       int32              `json:"state"`
		RevisionID  string             `json:"revision_id"`
		Description *string            `json:"description"`
		Labels      *map[string]string `json:"labels,omitempty"`
		SourceCode  string             `json:"source_code,omitempty"`
	} `json:"workflow"`
}

type RunJobEvent struct {
	Parent        string `json:"parent"`
	ExecutionName string `json:"executionName"`
	Argument      string `json:"argument,omitempty"`
}
