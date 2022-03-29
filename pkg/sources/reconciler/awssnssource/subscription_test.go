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

package awssnssource

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sns"
)

func TestErrors(t *testing.T) {
	genericErr := assert.AnError
	genericAWSErr := awserr.New("TestError", "an error", assert.AnError)
	genericK8SErr := &apierrors.StatusError{}

	t.Run("AWS error", func(t *testing.T) {
		assert.True(t, isAWSError(genericAWSErr))
		assert.True(t, isAWSError(fmt.Errorf("wrapped: %w", genericAWSErr)))
		assert.False(t, isAWSError(genericErr))
	})

	t.Run("denied", func(t *testing.T) {
		deniedErr := awserr.New(sns.ErrCodeAuthorizationErrorException, "an error", assert.AnError)

		reqFailErr := func(httpCode int) error {
			return awserr.NewRequestFailure(genericAWSErr, httpCode, "00000000-0000-...")
		}

		emptyStaticCredsErr := credentials.ErrStaticCredentialsEmpty

		assert.True(t, isDenied(deniedErr))
		assert.True(t, isDenied(fmt.Errorf("wrapped: %w", deniedErr)))
		assert.True(t, isDenied(reqFailErr(http.StatusUnauthorized)))
		assert.True(t, isDenied(emptyStaticCredsErr))
		assert.True(t, isDenied(fmt.Errorf("wrapped: %w", emptyStaticCredsErr)))
		assert.False(t, isDenied(genericAWSErr))
		assert.False(t, isDenied(genericErr))
		assert.False(t, isDenied(reqFailErr(http.StatusBadRequest)))
	})

	t.Run("not found", func(t *testing.T) {
		notFoundAWSErr := awserr.New(sns.ErrCodeNotFoundException, "an error", assert.AnError)
		notFoundK8SErr := apierrors.NewNotFound(schema.GroupResource{}, "")

		assert.True(t, isNotFound(notFoundAWSErr))
		assert.True(t, isNotFound(fmt.Errorf("wrapped: %w", notFoundAWSErr)))
		assert.True(t, isNotFound(notFoundK8SErr))
		assert.True(t, isNotFound(fmt.Errorf("wrapped: %w", notFoundK8SErr)))
		assert.False(t, isNotFound(genericAWSErr))
		assert.False(t, isNotFound(genericK8SErr))
		assert.False(t, isNotFound(genericErr))
	})

	t.Run("print error", func(t *testing.T) {
		awssErrWithExtra := awserr.NewRequestFailure(genericAWSErr, 400, "0123456789")

		assert.Equal(t, genericAWSErr.Error(), toErrMsg(awssErrWithExtra), "Extras not trimmed from AWS error")
		assert.Equal(t, genericErr.Error(), toErrMsg(genericErr), "Error was altered")
	})
}
