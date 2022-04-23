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
	"encoding/json"

	"knative.dev/pkg/apis"
)

// Validate implements apis.Validatable
func (s *CloudEventsSource) Validate(ctx context.Context) *apis.FieldError {
	return s.Spec.Validate(ctx).ViaField("spec")
}

// Validate CloudEventsSource spec
func (s *CloudEventsSourceSpec) Validate(ctx context.Context) *apis.FieldError {
	if s.Credentials == nil {
		return nil
	}

	return s.Credentials.Validate(ctx).ViaField("credentials")
}

func (c *HTTPCredentials) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError

	if len(c.BasicAuths) != 0 {
		if _, err := json.Marshal(c.BasicAuths); err != nil {
			errs = errs.Also(apis.ErrInvalidValue(
				"basic authentication parameter cannot be marshaled into JSON", "basicAuths", err.Error()))
		}
	}

	return errs
}
