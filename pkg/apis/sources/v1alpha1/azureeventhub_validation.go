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
		if s.Auth.SASToken.ConnectionString != nil {
			return nil
		}
		if s.Auth.SASToken.KeyName == nil {
			errs = errs.Also(apis.ErrMissingField("spec.sasToken.keyName"))
		}
		if s.Auth.SASToken.KeyValue == nil {
			errs = errs.Also(apis.ErrMissingField("spec.sasToken.keyValue"))
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
