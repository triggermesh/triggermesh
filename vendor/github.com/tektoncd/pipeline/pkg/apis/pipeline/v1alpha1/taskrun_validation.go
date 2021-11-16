/*
Copyright 2019 The Tekton Authors

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
	"context"
	"fmt"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/apis"
)

var _ apis.Validatable = (*TaskRun)(nil)

// Validate taskrun
func (tr *TaskRun) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(tr.GetObjectMeta()).ViaField("metadata"); err != nil {
		return err
	}
	if apis.IsInDelete(ctx) {
		return nil
	}
	return tr.Spec.Validate(ctx)
}

// Validate taskrun spec
func (ts *TaskRunSpec) Validate(ctx context.Context) *apis.FieldError {
	if equality.Semantic.DeepEqual(ts, &TaskRunSpec{}) {
		return apis.ErrMissingField("spec")
	}

	// can't have both taskRef and taskSpec at the same time
	if (ts.TaskRef != nil && ts.TaskRef.Name != "") && ts.TaskSpec != nil {
		return apis.ErrDisallowedFields("spec.taskref", "spec.taskspec")
	}

	// Check that one of TaskRef and TaskSpec is present
	if (ts.TaskRef == nil || (ts.TaskRef != nil && ts.TaskRef.Name == "")) && ts.TaskSpec == nil {
		return apis.ErrMissingField("spec.taskref.name", "spec.taskspec")
	}

	// Validate TaskSpec if it's present
	if ts.TaskSpec != nil {
		if err := ts.TaskSpec.Validate(ctx); err != nil {
			return err
		}
	}

	// Deprecated
	// check for input resources
	if ts.Inputs != nil {
		if err := ts.Inputs.Validate(ctx, "spec.Inputs"); err != nil {
			return err
		}
	}

	// Deprecated
	// check for output resources
	if ts.Outputs != nil {
		if err := ts.Outputs.Validate(ctx, "spec.Outputs"); err != nil {
			return err
		}
	}

	// Validate Resources
	if err := ts.Resources.Validate(ctx); err != nil {
		return err
	}

	if err := validateWorkspaceBindings(ctx, ts.Workspaces); err != nil {
		return err
	}
	if err := validateParameters("spec.inputs.params", ts.Params); err != nil {
		return err
	}

	if ts.Timeout != nil {
		// timeout should be a valid duration of at least 0.
		if ts.Timeout.Duration < 0 {
			return apis.ErrInvalidValue(fmt.Sprintf("%s should be >= 0", ts.Timeout.Duration.String()), "spec.timeout")
		}
	}

	return nil
}

func (i TaskRunInputs) Validate(ctx context.Context, path string) *apis.FieldError {
	if err := validatePipelineResources(ctx, i.Resources, fmt.Sprintf("%s.Resources.Name", path)); err != nil {
		return err
	}
	return validateParameters("spec.inputs.params", i.Params)
}

func (o TaskRunOutputs) Validate(ctx context.Context, path string) *apis.FieldError {
	return validatePipelineResources(ctx, o.Resources, fmt.Sprintf("%s.Resources.Name", path))
}

// validateWorkspaceBindings makes sure the volumes provided for the Task's declared workspaces make sense.
func validateWorkspaceBindings(ctx context.Context, wb []WorkspaceBinding) *apis.FieldError {
	seen := sets.NewString()
	for _, w := range wb {
		if seen.Has(w.Name) {
			return apis.ErrMultipleOneOf("spec.workspaces.name")
		}
		seen.Insert(w.Name)

		if err := w.Validate(ctx).ViaField("workspace"); err != nil {
			return err
		}
	}

	return nil
}

// validatePipelineResources validates that
//	1. resource is not declared more than once
//	2. if both resource reference and resource spec is defined at the same time
//	3. at least resource ref or resource spec is defined
func validatePipelineResources(ctx context.Context, resources []TaskResourceBinding, path string) *apis.FieldError {
	encountered := sets.NewString()
	for _, r := range resources {
		// We should provide only one binding for each resource required by the Task.
		name := strings.ToLower(r.Name)
		if encountered.Has(strings.ToLower(name)) {
			return apis.ErrMultipleOneOf(path)
		}
		encountered.Insert(name)
		// Check that both resource ref and resource Spec are not present
		if r.ResourceRef != nil && r.ResourceSpec != nil {
			return apis.ErrDisallowedFields(fmt.Sprintf("%s.ResourceRef", path), fmt.Sprintf("%s.ResourceSpec", path))
		}
		// Check that one of resource ref and resource Spec is present
		if (r.ResourceRef == nil || r.ResourceRef.Name == "") && r.ResourceSpec == nil {
			return apis.ErrMissingField(fmt.Sprintf("%s.ResourceRef", path), fmt.Sprintf("%s.ResourceSpec", path))
		}
		if r.ResourceSpec != nil && r.ResourceSpec.Validate(ctx) != nil {
			return r.ResourceSpec.Validate(ctx)
		}
	}

	return nil
}

// TODO(jasonhall): Share this with v1beta1/taskrun_validation.go
func validateParameters(path string, params []Param) *apis.FieldError {
	// Template must not duplicate parameter names.
	seen := sets.NewString()
	for _, p := range params {
		if seen.Has(strings.ToLower(p.Name)) {
			return apis.ErrMultipleOneOf(path)
		}
		seen.Insert(p.Name)
	}
	return nil
}
