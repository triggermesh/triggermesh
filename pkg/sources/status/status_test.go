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

package status

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

func TestExactReason(t *testing.T) {
	const fakeReason = "FakeReason"

	containerTestCases := []struct {
		name         string
		input        *corev1.ContainerStateWaiting
		expectReason string
	}{
		{
			name: "Generic error condition",
			input: &corev1.ContainerStateWaiting{
				Reason:  fakeReason,
				Message: "Some error message",
			},
			expectReason: fakeReason,
		},
		{
			name: "Container image error",
			input: &corev1.ContainerStateWaiting{
				Reason:  "ImagePullBackOff",
				Message: "Some error message",
			},
			expectReason: reasonBadContainerImage,
		},
		{
			name: "Runtime error",
			input: &corev1.ContainerStateWaiting{
				Reason:  "CrashLoopBackOff",
				Message: "Some error message",
			},
			expectReason: reasonAppRuntimeFailure,
		},
		{
			name: "Missing resource error",
			input: &corev1.ContainerStateWaiting{
				Reason:  "CreateContainerConfigError",
				Message: `foo "xyz" not found`,
			},
			expectReason: "MissingFoo",
		},
		{
			name: "Missing resource error with wrong reason",
			input: &corev1.ContainerStateWaiting{
				Reason:  fakeReason,
				Message: `foo "xyz" not found`,
			},
			expectReason: fakeReason,
		},
	}

	knativeTestCases := []struct {
		name         string
		input        *apis.Condition
		expectReason string
	}{
		{
			name: "Generic error condition",
			input: &apis.Condition{
				Reason:  fakeReason,
				Message: "Some error message",
			},
			expectReason: fakeReason,
		},
		{
			name: "Runtime error",
			input: &apis.Condition{
				Reason:  knRevisionFailedReason,
				Message: "Some error message: Container failed with: oops",
			},
			expectReason: reasonAppRuntimeFailure,
		},
		{
			name: "Runtime error with wrong reason",
			input: &apis.Condition{
				Reason:  fakeReason,
				Message: "Some error message: Container failed with: oops",
			},
			expectReason: fakeReason,
		},
		{
			name: "Container image error",
			input: &apis.Condition{
				Reason:  knRevisionFailedReason,
				Message: `Unable to fetch image "foo": some error`,
			},
			expectReason: reasonBadContainerImage,
		},
		{
			name: "Container image error with wrong reason",
			input: &apis.Condition{
				Reason:  fakeReason,
				Message: `Unable to fetch image "foo": some error`,
			},
			expectReason: fakeReason,
		},
		{
			name: "Missing resource error",
			input: &apis.Condition{
				Reason:  knRevisionFailedReason,
				Message: `foo "xyz" not found`,
			},
			expectReason: "MissingFoo",
		},
		{
			name: "Missing resource error with wrong reason",
			input: &apis.Condition{
				Reason:  fakeReason,
				Message: `foo "xyz" not found`,
			},
			expectReason: fakeReason,
		},
	}

	t.Run("Container", func(t *testing.T) {
		for _, tc := range containerTestCases {
			t.Run(tc.name, func(t *testing.T) {
				reason := ExactReason(tc.input)
				require.Equal(t, tc.expectReason, reason)
			})
		}
	})

	t.Run("Knative", func(t *testing.T) {
		for _, tc := range knativeTestCases {
			t.Run(tc.name, func(t *testing.T) {
				reason := ExactReason(tc.input)
				require.Equal(t, tc.expectReason, reason)
			})
		}
	})
}

func TestIsResourceMissingError(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectMissing bool
		expectType    string
	}{
		{
			name:          "Matching input format",
			input:         `failed to do xyz: secret "my-api-token" not found`,
			expectMissing: true,
			expectType:    "Secret",
		},
		{
			name:          "Matching input format without error decoration",
			input:         `secret "my-api-token" not found`,
			expectMissing: true,
			expectType:    "Secret",
		},
		{
			name:          "Matching input format with single trailing period",
			input:         `failed to do xyz: secret "my-api-token" not found.`,
			expectMissing: true,
			expectType:    "Secret",
		},
		{
			name:          "Non-matching input format",
			input:         `failed to do xyz: something "my-api-token" something`,
			expectMissing: false,
		},
		{
			name:          "No opening quote",
			input:         `failed to do xyz: secret my-api-token" not found`,
			expectMissing: false,
		},
		{
			name:          "No resource type",
			input:         `failed to do xyz: "my-api-token" not found`,
			expectMissing: false,
		},
		{
			name:          "Input starts with a quote",
			input:         `"my-api-token" not found`,
			expectMissing: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			missing, typ := isResourceMissingError(knRevisionFailedReason, tc.input)
			require.Equal(t, tc.expectMissing, missing, "Message describes a missing resource")
			require.Equal(t, tc.expectType, typ)
		})
	}
}
