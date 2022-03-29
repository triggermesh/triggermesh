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

package auth_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/triggermesh/triggermesh/pkg/sources/auth"
)

func TestPermanentCredentialsError(t *testing.T) {
	genericErr := assert.AnError
	permErr := NewPermanentCredentialsError(genericErr)

	assert.False(t, isPermanent(genericErr))
	assert.False(t, isPermanent(fmt.Errorf("wrapped: %w", genericErr)))

	assert.True(t, isPermanent(permErr))
	assert.True(t, isPermanent(fmt.Errorf("wrapped: %w", permErr)))

}

// isPermanent returns whether err implements PermanentCredentialsError.
func isPermanent(err error) bool {
	permErr := (PermanentCredentialsError)(nil)
	return errors.As(err, &permErr)
}
