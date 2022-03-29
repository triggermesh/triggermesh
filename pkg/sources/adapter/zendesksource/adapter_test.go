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

package zendesksource

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestStart verifies that a started adapter responds to cancelation.
func TestStart(t *testing.T) {
	const testTimeout = time.Second * 2
	testCtx, testCancel := context.WithTimeout(context.Background(), testTimeout)
	defer testCancel()

	a := &adapter{}

	// errCh receives the error value returned by the receiver after
	// termination. We leave it open to avoid panicking in case the
	// receiver returns after the timeout.
	errCh := make(chan error)

	// ctx gets canceled to cause a voluntary interruption of the receiver
	ctx, cancel := context.WithCancel(testCtx)
	go func() {
		errCh <- a.Start(ctx)
	}()
	cancel()

	select {
	case <-testCtx.Done():
		t.Errorf("Test timed out after %v", testTimeout)
	case err := <-errCh:
		assert.NoError(t, err, "Adapter returned an error")
	}
}
