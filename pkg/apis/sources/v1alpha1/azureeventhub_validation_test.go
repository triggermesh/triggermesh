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
	"testing"

	"github.com/stretchr/testify/assert"
	"knative.dev/pkg/apis"
)

func stringPtr(s string) *string {
	return &s
}

// Validate implements apis.Validatable.
func TestValidate(t *testing.T) {
	testCases := []struct {
		name        string
		source      AzureEventHubSource
		expectedErr error
	}{{
		name: "Valid auth with Connection String",
		source: AzureEventHubSource{
			Spec: AzureEventHubSourceSpec{
				Auth: AzureAuth{
					SASToken: &AzureSASToken{
						ConnectionString: stringPtr("foo"),
					},
				},
			},
		},
	}, {
		name: "Missing Hub name",
		source: AzureEventHubSource{
			Spec: AzureEventHubSourceSpec{
				HubNamespace: "baz",
				Auth: AzureAuth{
					SASToken: &AzureSASToken{
						KeyName:  stringPtr("foo"),
						KeyValue: stringPtr("bar"),
					},
				},
			},
		},
		expectedErr: apis.ErrMissingField("spec.hubName"),
	}, {
		name: "Missing multiple fields",
		source: AzureEventHubSource{
			Spec: AzureEventHubSourceSpec{
				HubName: "foo",
				Auth: AzureAuth{
					SASToken: &AzureSASToken{
						KeyName: stringPtr("bar"),
					},
				},
			},
		},
		expectedErr: apis.ErrMissingField("spec.hubNamespace", "spec.sasToken.keyValue"),
	}}

	for _, tc := range testCases {
		//nolint:scopelint
		t.Run(tc.name, func(t *testing.T) {
			err := tc.source.Validate(context.Background())
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
			}
		})
	}
}
