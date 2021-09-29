/*
Copyright 2020 TriggerMesh Inc.

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

// Validate implements apis.Validatable.
func (s *AzureEventHubSource) Validate(ctx context.Context) *apis.FieldError {
	err := s.Spec.ValidateSpec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// ValidateSpec validates Azure Event Hub spec parameters
func (s *AzureEventHubSourceSpec) ValidateSpec(_ context.Context) *apis.FieldError {
	var errs *apis.FieldError

	if s.Auth.SASToken != nil {
		if fieldIsSet(s.Auth.SASToken.ConnectionString) {
			return nil
		}
		if !fieldIsSet(s.Auth.SASToken.KeyName) {
			errs = errs.Also(apis.ErrMissingField("spec.auth.sasToken.keyName"))
		}
		if !fieldIsSet(s.Auth.SASToken.KeyValue) {
			errs = errs.Also(apis.ErrMissingField("spec.auth.sasToken.keyValue"))
		}
	}

	if s.HubName == "" {
		errs = errs.Also(apis.ErrMissingField("spec.hubName"))
	}
	if s.HubNamespace == "" {
		errs = errs.Also(apis.ErrMissingField("spec.hubNamespace"))
	}

	return errs
}

func fieldIsSet(f ValueFromField) bool {
	return f.Value != "" || f.ValueFromSecret != nil
}
