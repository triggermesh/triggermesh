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
	"context"
	"fmt"

	"knative.dev/pkg/apis"

	"github.com/triggermesh/triggermesh/pkg/routing/eventfilter/cel"
)

// Validate implements apis.Validatable
func (f *Filter) Validate(ctx context.Context) *apis.FieldError {
	return f.Spec.Validate(ctx).ViaField("spec")
}

// Validate implements apis.Validatable
func (fs *FilterSpec) Validate(ctx context.Context) *apis.FieldError {
	if fs.Expression == "" {
		return apis.ErrMissingField("Expression")
	}
	if fs.Sink == nil {
		return apis.ErrMissingField("Sink")
	}
	if _, err := cel.CompileExpression(fs.Expression); err != nil {
		return apis.ErrInvalidValue(fmt.Sprintf("Cannot compile expression: %v", err), "Expression")
	}
	return nil
}
