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

package awssqssource

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
	metricNameQueueCapacityProcess    = "queue_capacity_process"
	metricNameQueueCapacityDelete     = "queue_capacity_delete"
	metricNameMsgEnqueuedProcessCount = "message_enqueued_process_count"
	metricNameMsgDequeuedProcessCount = "message_dequeued_process_count"
	metricNameMsgEnqueuedDeleteCount  = "message_enqueued_delete_count"
	metricNameMsgDequeuedDeleteCount  = "message_dequeued_delete_count"
)

var (
	tagKeyResourceGroup = tag.MustNewKey(eventingmetrics.LabelResourceGroup)
	tagKeyNamespace     = tag.MustNewKey(eventingmetrics.LabelNamespaceName)
	tagKeyName          = tag.MustNewKey(eventingmetrics.LabelName)
)

// queueCapacityProcessM records the capacity of the processing queue.
var queueCapacityProcessM = stats.Int64(
	metricNameQueueCapacityProcess,
	"Number of items that can be buffered in the processing queue",
	stats.UnitDimensionless,
)

// queueCapacityDeleteM records the capacity of the deletion queue.
var queueCapacityDeleteM = stats.Int64(
	metricNameQueueCapacityDelete,
	"Number of items that can be buffered in the deletion queue",
	stats.UnitDimensionless,
)

// msgEnqueuedProcessCountM records the number of SQS messages that have been
// put onto the processing queue.
var msgEnqueuedProcessCountM = stats.Int64(
	metricNameMsgEnqueuedProcessCount,
	"Number of SQS messages that have been put onto the processing queue",
	stats.UnitDimensionless,
)

// msgDequeuedProcessCountM records the number of SQS messages that have been
// fetched from the processing queue.
var msgDequeuedProcessCountM = stats.Int64(
	metricNameMsgDequeuedProcessCount,
	"Number of SQS messages that have been fetched from the processing queue",
	stats.UnitDimensionless,
)

// msgEnqueuedDeleteCountM records the number of SQS messages that have been
// put onto the deletion queue.
var msgEnqueuedDeleteCountM = stats.Int64(
	metricNameMsgEnqueuedDeleteCount,
	"Number of SQS messages that have been put onto the deletion queue",
	stats.UnitDimensionless,
)

// msgDequeuedDeleteCountM records the number of SQS messages that have been
// fetched from the deletion queue.
var msgDequeuedDeleteCountM = stats.Int64(
	metricNameMsgDequeuedDeleteCount,
	"Number of SQS messages that have been fetched from the deletion queue",
	stats.UnitDimensionless,
)

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
			Measure:     queueCapacityProcessM,
			Description: queueCapacityProcessM.Description(),
			Aggregation: view.LastValue(),
			TagKeys:     tagKeys,
		},
		&view.View{
			Measure:     queueCapacityDeleteM,
			Description: queueCapacityDeleteM.Description(),
			Aggregation: view.LastValue(),
			TagKeys:     tagKeys,
		},
		&view.View{
			Measure:     msgEnqueuedProcessCountM,
			Description: msgEnqueuedProcessCountM.Description(),
			Aggregation: view.Count(),
			TagKeys:     tagKeys,
		},
		&view.View{
			Measure:     msgDequeuedProcessCountM,
			Description: msgDequeuedProcessCountM.Description(),
			Aggregation: view.Count(),
			TagKeys:     tagKeys,
		},
		&view.View{
			Measure:     msgEnqueuedDeleteCountM,
			Description: msgEnqueuedDeleteCountM.Description(),
			Aggregation: view.Count(),
			TagKeys:     tagKeys,
		},
		&view.View{
			Measure:     msgDequeuedDeleteCountM,
			Description: msgDequeuedDeleteCountM.Description(),
			Aggregation: view.Count(),
			TagKeys:     tagKeys,
		},
	)
	if err != nil {
		panic(fmt.Errorf("error registering OpenCensus stats view: %w", err))
	}
}

// statsReporter collects and reports stats about the event source.
type statsReporter struct {
	// context that holds pre-populated OpenCensus tags
	tagsCtx context.Context
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

// reportQueueCapacityProcess sets the value of queueCapacityProcessM.
func (r *statsReporter) reportQueueCapacityProcess(cap int) {
	metrics.Record(r.tagsCtx, queueCapacityProcessM.M(int64(cap)))
}

// reportQueueCapacityDelete sets the value of queueCapacityDeleteM.
func (r *statsReporter) reportQueueCapacityDelete(cap int) {
	metrics.Record(r.tagsCtx, queueCapacityDeleteM.M(int64(cap)))
}

// reportMessageEnqueuedProcessCount increments msgEnqueuedProcessCountM.
func (r *statsReporter) reportMessageEnqueuedProcessCount() {
	metrics.Record(r.tagsCtx, msgEnqueuedProcessCountM.M(1))
}

// reportMessageDequeuedProcessCount increments msgDequeuedProcessCountM.
func (r *statsReporter) reportMessageDequeuedProcessCount() {
	metrics.Record(r.tagsCtx, msgDequeuedProcessCountM.M(1))
}

// reportMessageEnqueuedDeleteCount increments msgEnqueuedDeleteCountM.
func (r *statsReporter) reportMessageEnqueuedDeleteCount() {
	metrics.Record(r.tagsCtx, msgEnqueuedDeleteCountM.M(1))
}

// reportMessageDequeuedDeleteCount increments msgDequeuedDeleteCountM.
func (r *statsReporter) reportMessageDequeuedDeleteCount() {
	metrics.Record(r.tagsCtx, msgDequeuedDeleteCountM.M(1))
}
