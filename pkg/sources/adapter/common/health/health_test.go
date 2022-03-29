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

package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	logtesting "knative.dev/pkg/logging/testing"
)

func TestHandler(t *testing.T) {
	currentHealthPort := healthPort
	defer func() {
		// reset port and handler state for other tests, or in case
		// this test is executed multiple times with -count
		healthPort = currentHealthPort
		defaultHandler = handler{}
	}()

	healthPort = getAvailablePort(t)
	healthURL := fmt.Sprintf("http://:%d%s", healthPort, healthPath)

	ctx, cancel := context.WithCancel(logtesting.TestContextWithLogger(t))
	defer cancel()

	done := make(chan struct{})
	go func() {
		Start(ctx)
		done <- struct{}{}
	}()

	// the server may need some time before it can serve the first request
	pollURL(t, healthURL)

	// not ready - expect HTTP error
	resp, err := http.Get(healthURL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	resp.Body.Close()

	MarkReady()

	// ready - expect HTTP success
	resp, err = http.Get(healthURL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	cancel()
	<-done
}

// getFreePort returns a TCP port that's guaranteed to be currently available
// on the local host.
func getAvailablePort(t *testing.T) uint16 {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("Failed to obtain a new Listener:", err)
	}

	defer l.Close()

	return uint16(l.Addr().(*net.TCPAddr).Port)
}

// pollURL polls the given URL until a HTTP response is received.
func pollURL(t *testing.T, url string) {
	t.Helper()

	var pollErr error

	for i := 0; i < 10; i++ {
		_, pollErr = http.Get(url)
		if pollErr == nil {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("Health server didn't start. Last client error:", pollErr)
}
