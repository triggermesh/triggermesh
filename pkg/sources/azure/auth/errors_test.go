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

package auth_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/triggermesh/triggermesh/pkg/sources/azure/auth"
)

func TestFatalCredentialsError(t *testing.T) {
	genericErr := assert.AnError
	fatalErr := NewFatalCredentialsError(genericErr)

	assert.False(t, isFatal(genericErr))
	assert.False(t, isFatal(fmt.Errorf("wrapped: %w", genericErr)))

	assert.True(t, isFatal(fatalErr))
	assert.True(t, isFatal(fmt.Errorf("wrapped: %w", fatalErr)))

}

// isFatal returns whether err implements FatalCredentialsError.
func isFatal(err error) bool {
	fatalErr := (FatalCredentialsError)(nil)
	return errors.As(err, &fatalErr)
}
