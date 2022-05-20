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

package metrics

import (
	"testing"

	"knative.dev/pkg/metrics/metricstest"

	// Essential. Initializes a Prometheus metrics exporter for tests.
	_ "knative.dev/pkg/metrics/testing"

	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// ResetMetrics resets the global state of OpenCensus metrics.
// Must be called between unit tests.
func ResetMetrics(t *testing.T) {
	t.Helper()

	UnregisterMetrics()

	metrics.MustRegisterEventProcessingStatsView()

	metricstest.AssertNoMetric(t,
		"event_processing_success_count",
		"event_processing_error_count",
		"event_processing_latencies",
	)
}

// UnregisterMetrics unregisters the metrics that were registered in the global
// state of OpenCensus.
// Can be used instead of ResetMetrics to avoid panics in tests that already
// call metrics.MustRegisterEventProcessingStatsView.
func UnregisterMetrics() {
	metricstest.Unregister(
		"event_processing_success_count",
		"event_processing_error_count",
		"event_processing_latencies",
	)
}
