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

package common

import (
	"context"
	"math"
	"sync/atomic"
	"time"
)

// default values for backoff
const (
	expFactor = 2

	defaultMinBackoff = 1 * time.Second
	defaultMaxBackoff = 32 * time.Second
)

// Backoff provides a simple exponential backoff mechanism.
type Backoff struct {
	step     *int32
	factor   float64
	min, max time.Duration
}

// RunFunc is a user function that polls data from a source and sends it as a
// CloudEvent to a sink.
// RunFunc must return (bool, error) values where bool is true if the poll
// backoff duration must be reset, and error is the result of the function's
// execution.
type RunFunc func(context.Context) (bool /*reset*/, error /*exit*/)

// NewBackoff accepts optional values for minimum and maximum wait period and
// returns a new instance of Backoff.
func NewBackoff(args ...time.Duration) *Backoff {
	backoff := &Backoff{
		step:   new(int32),
		factor: expFactor,
		min:    defaultMinBackoff,
		max:    defaultMaxBackoff,
	}

	switch len(args) {
	case 1:
		if args[0] <= backoff.max {
			backoff.min = args[0]
		}
	case 2:
		if args[0] <= args[1] {
			backoff.min = args[0]
			backoff.max = args[1]
		}
	}

	return backoff
}

// Duration returns the exponential backoff duration calculated for the current step.
func (b *Backoff) Duration() time.Duration {
	dur := time.Duration(float64(b.min)*math.Pow(b.factor, float64(atomic.LoadInt32(b.step))) - float64(b.min))

	switch {
	case dur < b.min:
		atomic.AddInt32(b.step, 1)
		return b.min
	case dur > b.max:
		return b.max
	default:
		atomic.AddInt32(b.step, 1)
		return dur
	}
}

// Reset sets step counter to zero.
func (b *Backoff) Reset() {
	atomic.StoreInt32(b.step, 0)
}

// Run is a blocking function that executes RunFunc until stopCh receives a
// value or fn returns an error.
func (b *Backoff) Run(stopCh <-chan struct{}, fn RunFunc) error {
	timer := time.NewTimer(0)
	defer timer.Stop()

	// FIXME(antoineco): never canceled until stopCh receives a value,
	// after which the fn is never invoked again, so ctx does effectively
	// nothing.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-stopCh:
			return nil

		case <-timer.C:
			reset, err := fn(ctx)
			if err != nil {
				return err
			}

			if reset {
				b.Reset()
			}
			timer.Reset(b.Duration())
		}
	}
}
