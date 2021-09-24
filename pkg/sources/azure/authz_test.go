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

package azure

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyCredentialsError(t *testing.T) {
	genericErr := assert.AnError
	ecErr := emptyCredentialsError{e: genericErr}

	assert.False(t, isEmptyCreds(genericErr))
	assert.False(t, isEmptyCreds(fmt.Errorf("wrapped: %w", genericErr)))

	assert.True(t, isEmptyCreds(ecErr))
	assert.True(t, isEmptyCreds(fmt.Errorf("wrapped: %w", ecErr)))

}

// isEmptyCreds returns whether err implements the IsEmptyCredentials error behaviour.
func isEmptyCreds(err error) bool {
	ecErr := (interface{ IsEmptyCredentials() })(nil)
	return errors.As(err, &ecErr)
}
