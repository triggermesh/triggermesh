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

package metrics_test

import (
	"testing"
	"time"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/metrics/metricstest"

	metricstesting "github.com/triggermesh/triggermesh/pkg/metrics/testing"

	. "github.com/triggermesh/triggermesh/pkg/metrics"
)

func TestEventProcessingStatsReporter(t *testing.T) {
	const (
		tRg   = "foos.fake.example.com"
		tNs   = "test-ns"
		tName = "test"
	)

	testMetricTags := &pkgadapter.MetricTag{
		ResourceGroup: tRg,
		Namespace:     tNs,
		Name:          tName,
	}

	st := MustNewEventProcessingStatsReporter(testMetricTags)

	wantCommonTags := map[string]string{
		"resource_group": tRg,
		"namespace_name": tNs,
		"name":           tName,
	}

	t.Run("record without tags", func(t *testing.T) {
		metricstesting.ResetMetrics(t)

		st.ReportProcessingSuccess()
		st.ReportProcessingError(true)
		st.ReportProcessingLatency(12 * time.Millisecond)

		metricstest.CheckCountData(t,
			"event_processing_success_count",
			wantCommonTags,
			1,
		)

		metricstest.CheckCountData(t,
			"event_processing_error_count",
			appendTags(wantCommonTags, map[string]string{
				"user_managed": "true",
			}),
			1,
		)

		metricstest.CheckDistributionCount(t,
			"event_processing_latencies",
			wantCommonTags,
			1,
		)
	})

	t.Run("record with tags", func(t *testing.T) {
		metricstesting.ResetMetrics(t)

		const tEventType = "test.type.v0"
		tagEventType := TagEventType(tEventType)

		st.ReportProcessingSuccess(tagEventType)
		st.ReportProcessingError(true, tagEventType)
		st.ReportProcessingLatency(12*time.Millisecond, tagEventType)

		wantCommonTagsWithEventType := appendTags(wantCommonTags, map[string]string{
			"event_type": tEventType,
		})

		metricstest.CheckCountData(t,
			"event_processing_success_count",
			wantCommonTagsWithEventType,
			1,
		)

		metricstest.CheckCountData(t,
			"event_processing_error_count",
			appendTags(wantCommonTagsWithEventType, map[string]string{
				"user_managed": "true",
			}),
			1,
		)

		metricstest.CheckDistributionCount(t,
			"event_processing_latencies",
			wantCommonTagsWithEventType,
			1,
		)
	})

	t.Run("latency distribution", func(t *testing.T) {
		metricstesting.ResetMetrics(t)

		st.ReportProcessingLatency(12 * time.Millisecond)
		st.ReportProcessingLatency(374 * time.Millisecond)
		st.ReportProcessingLatency(2250 * time.Millisecond)

		metricstest.CheckDistributionData(t,
			"event_processing_latencies",
			wantCommonTags,
			3,
			12.0,
			2250.0,
		)
	})
}

// appendTags returns a copy of the given metrics tags with extra key/values inserted.
func appendTags(tags, kvs map[string]string) map[string]string {
	tagsCpy := make(map[string]string, len(tags)+len(kvs))

	for k, v := range tags {
		tagsCpy[k] = v
	}
	for k, v := range kvs {
		tagsCpy[k] = v
	}

	return tagsCpy
}
