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

package ratelimiter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter(t *testing.T) {
	ctx := context.Background()

	// allow no more than 1 request per second.
	rl, err := New(1)

	require.NoError(t, err, "Error creating rate limiter")
	defer rl.Close(ctx)

	// first request must be allowed.
	ok, _, err := rl.Allow(ctx, nil)
	require.NoError(t, err, "Error calling rate limiter Allow method")
	assert.True(t, ok, "first request was not accepted")

	// requests spaced more than 1 second must make it through.
	time.Sleep(time.Second)
	ok, _, err = rl.Allow(ctx, nil)
	require.NoError(t, err, "Error calling rate limiter Allow method")
	assert.True(t, ok, "request spaced more than 1 second was not accepted")

	// we expect the test environment to make this call almost immediately and
	// far from the second range. If so, the Allow method should fail.
	ok, _, err = rl.Allow(ctx, nil)
	require.NoError(t, err, "Error calling rate limiter Allow method")
	assert.False(t, ok, "request spaced less than 1 second was accepted")
}
