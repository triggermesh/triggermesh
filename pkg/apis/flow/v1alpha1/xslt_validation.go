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

	"knative.dev/pkg/apis"
)

// Validate implements apis.Validatable
func (t *XSLTTransformation) Validate(ctx context.Context) *apis.FieldError {
	return t.Spec.Validate(ctx).ViaField("spec")
}

// Validate XSLT spec
func (s *XSLTTransformationSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError

	if (s.AllowPerEventXSLT == nil || !*s.AllowPerEventXSLT) && !s.XSLT.IsInformed() {
		errs = errs.Also(apis.ErrGeneric("when XSLT is empty, per event XSLT must be allowed", "allowPerEventXSLT", "xslt"))
	}

	if err := s.XSLT.Validate(ctx); err != nil {
		errs = errs.Also(err.ViaField("XSLT"))
	}

	return errs
}
