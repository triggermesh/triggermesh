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
	"net/http"
	"time"

	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/sethvargo/go-limiter"
	"github.com/sethvargo/go-limiter/memorystore"
)

const (
	// token to be used globally for every request.
	globalToken = "global"
)

type rateLimiter struct {
	store limiter.Store
}

// New creates a new rate limiter.
func New(rps uint64) (cehttp.RateLimiter, error) {
	if store, err := memorystore.New(&memorystore.Config{
		Tokens:   rps,
		Interval: time.Second,
	}); err != nil {
		return nil, err
	} else {
		return &rateLimiter{
			store: store,
		}, nil
	}
}

// Allow checks if a request is allowed to pass the rate limiter filter.
func (rl *rateLimiter) Allow(ctx context.Context, _ *http.Request) (ok bool, reset uint64, err error) {
	_, _, reset, ok, err = rl.store.Take(ctx, globalToken)
	return ok, reset, err
}

// Close cleans up rate limiter resources.
func (rl *rateLimiter) Close(ctx context.Context) error {
	return rl.store.Close(ctx)
}
