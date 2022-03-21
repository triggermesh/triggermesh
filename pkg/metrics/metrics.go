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

// Package metrics contains interfaces for reporting OpenCensus stats to a
// metrics backend, such as Prometheus.
package metrics

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	eventingmetrics "knative.dev/eventing/pkg/metrics"
	"knative.dev/pkg/metrics"
)

const (
	metricNameEventProcessingSuccessCount = "event_processing_success_count"
	metricNameEventProcessingErrorCount   = "event_processing_error_count"
	metricNameEventProcessingLatencies    = "event_processing_latencies"

	// Conveys whether the delivery of the error returned as the result of
	// a failed event processing is user-managed, as opposed to managed by
	// Knative (retries, dead-letter queue).
	labelUserManagedErr = "user_managed"
)

var (
	tagKeyResourceGroup  = tag.MustNewKey(eventingmetrics.LabelResourceGroup)
	tagKeyNamespace      = tag.MustNewKey(eventingmetrics.LabelNamespaceName)
	tagKeyName           = tag.MustNewKey(eventingmetrics.LabelName)
	tagKeyEventType      = tag.MustNewKey(eventingmetrics.LabelEventType)
	tagKeyEventSource    = tag.MustNewKey(eventingmetrics.LabelEventSource)
	tagKeyUserManagedErr = tag.MustNewKey(labelUserManagedErr)
)

// eventProcessingSuccessCountM is a measure of the number of events that were
// successfully processed by a component.
var eventProcessingSuccessCountM = stats.Int64(
	metricNameEventProcessingSuccessCount,
	"Number of events successfully processed by the CloudEvents handler",
	stats.UnitDimensionless,
)

// eventProcessingSuccessCountM is a measure of the number of events that were
// unsuccessfully processed by a component.
var eventProcessingErrorCountM = stats.Int64(
	metricNameEventProcessingErrorCount,
	"Number of events unsuccessfully processed by the CloudEvents handler",
	stats.UnitDimensionless,
)

// eventProcessingLatenciesM is a measure of the time spent by a component
// processing events.
var eventProcessingLatenciesM = stats.Int64(
	metricNameEventProcessingLatencies,
	"Time spent in the CloudEvents handler processing events",
	stats.UnitMilliseconds,
)

// MustRegisterEventProcessingStatsView registers an OpenCensus stats view for
// metrics related to events processing, and panics in case of error.
func MustRegisterEventProcessingStatsView() {
	commonTagKeys := []tag.Key{
		tagKeyResourceGroup,
		tagKeyNamespace,
		tagKeyName,
		tagKeyEventType,
		tagKeyEventSource,
	}

	err := view.Register(
		&view.View{
			Measure:     eventProcessingSuccessCountM,
			Description: eventProcessingSuccessCountM.Description(),
			Aggregation: view.Count(),
			TagKeys:     commonTagKeys,
		},
		&view.View{
			Measure:     eventProcessingErrorCountM,
			Description: eventProcessingErrorCountM.Description(),
			Aggregation: view.Count(),
			TagKeys: append(commonTagKeys,
				tagKeyUserManagedErr,
			),
		},
		&view.View{
			Measure:     eventProcessingLatenciesM,
			Description: eventProcessingLatenciesM.Description(),
			Aggregation: view.Distribution(metrics.Buckets125(1, 10000)...), // 1,2,5,10,20,50,100,200,500,1000,2000,5000,10000
			TagKeys:     commonTagKeys,
		},
	)
	if err != nil {
		panic(fmt.Errorf("error registering OpenCensus stats view: %w", err))
	}
}

// EventProcessingStatsReporter collects and reports stats about the processing of CloudEvents.
type EventProcessingStatsReporter struct {
	// context that holds pre-populated OpenCensus tags
	tagsCtx context.Context
}

// MustNewEventProcessingStatsReporter returns a new EventProcessingStatsReporter
// initialized with the given tags and panics in case of error.
func MustNewEventProcessingStatsReporter(tags *pkgadapter.MetricTag) *EventProcessingStatsReporter {
	ctx, err := tag.New(context.Background(),
		tag.Insert(tagKeyResourceGroup, tags.ResourceGroup),
		tag.Insert(tagKeyNamespace, tags.Namespace),
		tag.Insert(tagKeyName, tags.Name),
	)
	if err != nil {
		panic(fmt.Errorf("error creating OpenCensus tags: %w", err))
	}

	return &EventProcessingStatsReporter{
		tagsCtx: ctx,
	}
}

// ReportProcessingSuccess increments eventProcessingSuccessCountM.
func (r *EventProcessingStatsReporter) ReportProcessingSuccess(tms ...tag.Mutator) {
	tagsCtx, _ := tag.New(r.tagsCtx, tms...)
	metrics.Record(tagsCtx, eventProcessingSuccessCountM.M(1))
}

// ReportProcessingError increments eventProcessingErrorCountM.
func (r *EventProcessingStatsReporter) ReportProcessingError(userManaged bool, tms ...tag.Mutator) {
	tms = append(tms,
		tag.Insert(tagKeyUserManagedErr, strconv.FormatBool(userManaged)),
	)

	tagsCtx, _ := tag.New(r.tagsCtx, tms...)
	metrics.Record(tagsCtx, eventProcessingErrorCountM.M(1))
}

// ReportProcessingLatency records in eventProcessingLatenciesM the processing
// duration of an event.
func (r *EventProcessingStatsReporter) ReportProcessingLatency(d time.Duration, tms ...tag.Mutator) {
	tagsCtx, _ := tag.New(r.tagsCtx, tms...)
	metrics.Record(tagsCtx, eventProcessingLatenciesM.M(d.Milliseconds()))
}

// TagEventType returns a tag mutator that injects the value of the
// "event_type" tag.
func TagEventType(val string) tag.Mutator {
	return tag.Insert(tagKeyEventType, val)
}

// TagEventSource returns a tag mutator that injects the value of the
// "event_source" tag.
func TagEventSource(val string) tag.Mutator {
	return tag.Insert(tagKeyEventSource, val)
}
