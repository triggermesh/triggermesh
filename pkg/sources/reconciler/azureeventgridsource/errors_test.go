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

package azureeventgridsource

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

func TestErrors(t *testing.T) {
	genericErr := assert.AnError
	genericAzureDetailedErr := autorest.DetailedError{}
	genericAzureReqErr := &azure.RequestError{}

	azureDetailedErr := func(httpCode int) autorest.DetailedError {
		return autorest.DetailedError{
			StatusCode: httpCode,
		}
	}

	azureReqErr := func(httpCode int) *azure.RequestError {
		return &azure.RequestError{
			DetailedError: azureDetailedErr(httpCode),
		}
	}

	t.Run("denied", func(t *testing.T) {
		assert.True(t, isDenied(azureDetailedErr(http.StatusUnauthorized)))
		assert.True(t, isDenied(fmt.Errorf("wrapped: %w", azureDetailedErr(http.StatusUnauthorized))))
		assert.True(t, isDenied(azureReqErr(http.StatusUnauthorized)))
		assert.True(t, isDenied(fmt.Errorf("wrapped: %w", azureReqErr(http.StatusUnauthorized))))
		assert.False(t, isDenied(genericAzureDetailedErr))
		assert.False(t, isDenied(genericAzureReqErr))
		assert.False(t, isDenied(genericErr))
		assert.False(t, isDenied(azureDetailedErr(http.StatusBadRequest)))
		assert.False(t, isDenied(azureReqErr(http.StatusBadRequest)))
	})

	t.Run("not found", func(t *testing.T) {
		assert.True(t, isNotFound(azureDetailedErr(http.StatusNotFound)))
		assert.True(t, isNotFound(fmt.Errorf("wrapped: %w", azureDetailedErr(http.StatusNotFound))))
		assert.False(t, isNotFound(genericAzureDetailedErr))
		assert.False(t, isNotFound(genericAzureReqErr))
		assert.False(t, isNotFound(genericErr))
		assert.False(t, isNotFound(azureReqErr(http.StatusNotFound)))
		assert.False(t, isNotFound(fmt.Errorf("wrapped: %w", azureReqErr(http.StatusNotFound))))
		assert.False(t, isNotFound(azureDetailedErr(http.StatusBadRequest)))
		assert.False(t, isNotFound(azureReqErr(http.StatusBadRequest)))
	})

	t.Run("print error", func(t *testing.T) {
		azureDetailedErrWithStatusCode := genericAzureDetailedErr
		azureDetailedErrWithStatusCode.Original = genericErr

		azureReqErrWithStatusCode := genericAzureReqErr
		azureReqErrWithStatusCode.Original = genericErr

		assert.Equal(t, genericErr.Error(), toErrMsg(azureDetailedErrWithStatusCode), "Extras not trimmed from Azure error")
		assert.Equal(t, genericErr.Error(), toErrMsg(azureReqErrWithStatusCode), "Extras not trimmed from Azure error")
		assert.Equal(t, genericErr.Error(), toErrMsg(genericErr), "Error was altered")
	})
}
