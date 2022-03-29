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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBackoff(t *testing.T) {
	testCases := map[string]struct {
		wantMin, wantMax time.Duration
		gotMin, gotMax   time.Duration
	}{
		"defaults": {
			gotMin: defaultMinBackoff,
			gotMax: defaultMaxBackoff,
		},
		"all correct": {
			wantMin: time.Second,
			gotMin:  time.Second,
			wantMax: time.Minute,
			gotMax:  time.Minute,
		},
		"min > default max": {
			wantMin: time.Hour,
			gotMin:  defaultMinBackoff,
			gotMax:  defaultMaxBackoff,
		},
		"min > max": {
			wantMin: time.Hour,
			gotMin:  defaultMinBackoff,
			wantMax: time.Minute,
			gotMax:  defaultMaxBackoff,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			bo := NewBackoff()
			switch {
			case tc.wantMin != 0 && tc.wantMax != 0:
				bo = NewBackoff(tc.wantMin, tc.wantMax)
			case tc.wantMin != 0:
				bo = NewBackoff(tc.wantMin)
			}
			assert.Equal(t, bo.min, tc.gotMin, "backoff min duration has unexpected value")
			assert.Equal(t, bo.max, tc.gotMax, "backoff max duration has unexpected value")
		})
	}
}

func TestDuration(t *testing.T) {
	testCases := map[string]struct {
		step              int32
		min, max, wantDur time.Duration
	}{
		"first step": {
			step:    0,
			min:     time.Second,
			max:     time.Minute,
			wantDur: time.Second, // = min
		},
		"third step": {
			step:    2,
			min:     time.Second,
			wantDur: 3 * time.Second, // = 1s * factor(=2)^step(=2) - 1s
		},
		"tenth step": {
			step:    9,
			min:     time.Second,
			max:     time.Minute,
			wantDur: time.Minute, // = max
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			bo := NewBackoff()
			switch {
			case tc.min != 0 && tc.max != 0:
				bo = NewBackoff(tc.min, tc.max)
			case tc.min != 0:
				bo = NewBackoff(tc.min)
			}

			atomic.StoreInt32(bo.step, tc.step)
			assert.Equal(t, tc.wantDur, bo.Duration())
		})
	}
}

func TestRun(t *testing.T) {
	testCases := map[string]struct {
		fn         func(context.Context) (bool, error)
		waitReturn bool
		err        error
	}{
		"force stop with no err": {
			fn: func(ctx context.Context) (bool, error) {
				// never terminates by itself
				return true, nil
			},
			waitReturn: false,
		},
		"fn returns err": {
			fn: func(ctx context.Context) (bool, error) {
				return false, assert.AnError
			},
			waitReturn: true,
			err:        assert.AnError,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			errCh := make(chan error)
			defer close(errCh)

			stopCh := make(chan struct{})

			t.Run(name, func(t *testing.T) {
				bo := NewBackoff()
				go func() {
					err := bo.Run(stopCh, tc.fn)
					errCh <- err
				}()

				if tc.waitReturn {
					defer close(stopCh)
				} else {
					close(stopCh)
				}

				err := <-errCh
				if tc.err != nil {
					assert.EqualError(t, err, tc.err.Error())
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}

	t.Run("fn fails several times then resets backoff duration", func(t *testing.T) {
		const minBackoff = 1 * time.Millisecond
		const timesRunFnBeforeReset = 3

		var returnVal bool

		// latch is used to keep the function executions under control
		// of the main test logic.
		latch := make(chan struct{})

		fn := func(ctx context.Context) (bool, error) {
			latch <- struct{}{}
			defer func() { latch <- struct{}{} }()

			return returnVal, nil
		}

		bo := NewBackoff(minBackoff)

		stopCh := make(chan struct{})
		errCh := make(chan error)
		defer close(errCh)

		go func() {
			errCh <- bo.Run(stopCh, fn)
		}()

		// cause timesRunFnBeforeReset fn executions
		for i := 0; i < timesRunFnBeforeReset; i++ {
			<-latch
			<-latch
		}
		durationBeforeReset := bo.Duration()

		returnVal = true // next run causes a reset
		<-latch
		<-latch

		close(stopCh)
		assert.NoError(t, <-errCh)
		durationAfterReset := bo.Duration()

		assert.Less(t, int64(durationAfterReset), int64(durationBeforeReset),
			"Duration after reset is not less then before reset")

		assert.Equal(t, durationAfterReset, minBackoff, "Backoff wasn't reset to its min value")
	})
}
