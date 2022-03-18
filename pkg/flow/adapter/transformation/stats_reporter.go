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

package transformation

import (
	"context"
	"fmt"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	eventingmetrics "knative.dev/eventing/pkg/metrics"
	"knative.dev/pkg/metrics"
)

const (
	metricNameEventsProcessingLatencies = "events_processing_latencies"
	metricNameEventsProcessingCount     = "events_processing_count"
	metricNameEventsProcessingErrs      = "events_processing_errors"
)

var (
	tagKeyResourceGroup = tag.MustNewKey(eventingmetrics.LabelResourceGroup)
	tagKeyNamespace     = tag.MustNewKey(eventingmetrics.LabelNamespaceName)
	tagKeyName          = tag.MustNewKey(eventingmetrics.LabelName)
)

// eventsProcessingLatencies records the latencies of events processing.
var eventsProcessingLatencies = stats.Int64(
	metricNameEventsProcessingLatencies,
	"The time spent on the events transformation",
	stats.UnitMilliseconds,
)

// eventsProcessingLatencies records the count of events transformation.
var eventsProcessingCount = stats.Int64(
	metricNameEventsProcessingCount,
	"Number of transformed events",
	stats.UnitDimensionless,
)

// eventsProcessingLatencies records the count of events processing errors.
var eventsProcessingErrs = stats.Int64(
	metricNameEventsProcessingErrs,
	"Number of times adapter failed to send the event",
	stats.UnitDimensionless,
)

// statsReporter collects and reports stats about the event source.
type statsReporter struct {
	// context that holds pre-populated OpenCensus tags
	tagsCtx context.Context
}

// mustRegisterStatsView registers an OpenCensus stats view for the source's
// metrics and panics in case of error.
func mustRegisterStatsView() {
	tagKeys := []tag.Key{
		tagKeyResourceGroup,
		tagKeyNamespace,
		tagKeyName,
	}

	err := view.Register(
		&view.View{
			Measure:     eventsProcessingLatencies,
			Description: eventsProcessingLatencies.Description(),
			Aggregation: view.Distribution(metrics.Buckets125(1, 1000)...), // 1, 2, 5, 10, 20, 50, 100, 500, 1000, 5000, 10000
			TagKeys:     tagKeys,
		},
		&view.View{
			Measure:     eventsProcessingCount,
			Description: eventsProcessingCount.Description(),
			Aggregation: view.Count(),
			TagKeys:     tagKeys,
		},
		&view.View{
			Measure:     eventsProcessingErrs,
			Description: eventsProcessingErrs.Description(),
			Aggregation: view.Count(),
			TagKeys:     tagKeys,
		},
	)
	if err != nil {
		panic(fmt.Errorf("error registering OpenCensus stats view: %w", err))
	}
}

// mustNewStatsReporter returns a new statsReporter initialized with the given
// tags and panics in case of error.
func mustNewStatsReporter(tags *pkgadapter.MetricTag) *statsReporter {
	ctx, err := tag.New(context.Background(),
		tag.Insert(tagKeyResourceGroup, tags.ResourceGroup),
		tag.Insert(tagKeyNamespace, tags.Namespace),
		tag.Insert(tagKeyName, tags.Name),
	)
	if err != nil {
		panic(fmt.Errorf("error creating OpenCensus tags: %w", err))
	}

	return &statsReporter{
		tagsCtx: ctx,
	}
}

// reportEventProcessingTime sets the value of eventsProcessingLatencies.
func (r *statsReporter) reportEventProcessingTime(duration int64) {
	metrics.Record(r.tagsCtx, eventsProcessingLatencies.M(duration))
}

// reportEventProcessingCount sets the value of eventsProcessingCount.
func (r *statsReporter) reportEventProcessingCount() {
	metrics.Record(r.tagsCtx, eventsProcessingCount.M(1))
}

// reportEventProcessingCount sets the value of eventsProcessingErrs.
func (r *statsReporter) reportEventProcessingError() {
	metrics.Record(r.tagsCtx, eventsProcessingErrs.M(1))
}
