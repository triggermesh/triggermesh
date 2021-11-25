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
	"context"

	"knative.dev/pkg/apis"
)

// Validate implements apis.Validatable
func (x *XsltTransform) Validate(ctx context.Context) *apis.FieldError {
	return x.Spec.Validate(ctx).ViaField("spec")
}

// Validates XSLT spec
func (xs *XsltTransformSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError
	if xs.AllowPerEventXSLT != nil && !*xs.AllowPerEventXSLT && !xs.XSLT.IsInformed() {
		errs = errs.Also(apis.ErrMissingOneOf("If XSLT is not allowed at each event payload, the XSLT must be present"))
	}

	if err := xs.XSLT.Validate(ctx); err != nil {
		errs = err.ViaField("XSLT")
	}

	return errs
}
