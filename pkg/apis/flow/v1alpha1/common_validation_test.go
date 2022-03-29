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
	"testing"

	"github.com/stretchr/testify/assert"
	"knative.dev/pkg/apis"
)

var (
	errVffTooMany               = apis.ErrMultipleOneOf("value", "valueFromSecret", "valueFromConfigMap")
	errVffMissingSecretField    = apis.ErrMissingField("name", "key").ViaField("ValueFromSecret")
	errVffMissingConfigMapField = apis.ErrMissingField("name", "key").ViaField("ValueFromConfigMap")
)

func TestValueFromFieldValidate(t *testing.T) {
	testCases := map[string]struct {
		vff         *ValueFromField
		expectError *apis.FieldError
	}{
		"value, ok": {
			vff:         valueFromField(vffWithValue(tValue)),
			expectError: nil,
		},
		"secret, ok": {
			vff:         valueFromField(vffWithSecret(tName, tKey)),
			expectError: nil,
		},
		"configmap, ok": {
			vff:         valueFromField(vffWithConfigMap(tName, tKey)),
			expectError: nil,
		},
		"nil, ok": {
			vff:         nil,
			expectError: nil,
		},
		"empty value, ok": {
			vff:         valueFromField(),
			expectError: nil,
		},
		"value and secret, fail": {
			vff:         valueFromField(vffWithValue(tValue), vffWithSecret(tName, tKey)),
			expectError: errVffTooMany,
		},
		"value and configmap, fail": {
			vff:         valueFromField(vffWithValue(tValue), vffWithConfigMap(tName, tKey)),
			expectError: errVffTooMany,
		},
		"secret and configmap, fail": {
			vff:         valueFromField(vffWithConfigMap(tName, tKey), vffWithSecret(tName, tKey)),
			expectError: errVffTooMany,
		},
		"value, secret and configmap, fail": {
			vff:         valueFromField(vffWithValue(tValue), vffWithConfigMap(tName, tKey), vffWithSecret(tName, tKey)),
			expectError: errVffTooMany,
		},
		"secret lacks name, fail": {
			vff:         valueFromField(vffWithSecret("", tKey)),
			expectError: errVffMissingSecretField,
		},
		"secret lacks key, fail": {
			vff:         valueFromField(vffWithSecret(tName, "")),
			expectError: errVffMissingSecretField,
		},
		"configmap lacks name, fail": {
			vff:         valueFromField(vffWithConfigMap("", tKey)),
			expectError: errVffMissingConfigMapField,
		},
		"configmap lacks key, fail": {
			vff:         valueFromField(vffWithConfigMap(tName, "")),
			expectError: errVffMissingConfigMapField,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expectError, tc.vff.Validate(context.Background()))
		})
	}
}
