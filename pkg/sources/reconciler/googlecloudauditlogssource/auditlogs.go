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

package googlecloudauditlogssource

import (
	"context"
	"fmt"
	"hash/crc32"
	"strconv"

	"cloud.google.com/go/logging/logadmin"
	"cloud.google.com/go/pubsub"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/event"
	"github.com/triggermesh/triggermesh/pkg/reconciler/skip"
)

const (
	publisherRole = "roles/pubsub.publisher"
)

func ReconcileSink(ctx context.Context, lacli *logadmin.Client, pscli *pubsub.Client, topicResName *v1alpha1.GCloudResourceName) error {
	if skip.Skip(ctx) {
		return nil
	}

	sink, err := ensureSinkCreated(ctx, lacli, topicResName)
	if err != nil {
		return fmt.Errorf("failed to create audit log sink: %w", err)
	}
	err = ensureSinkIsPublisher(ctx, sink, pscli, topicResName)
	if err != nil {
		return fmt.Errorf("failed to ensure sink has pubsub.publisher permission on source topic: %w", err)
	}
	return nil
}

// Ensures that the Audit Logs sink has been created.
// Required permissions:
// - logging.sinks.get
// - logging.sinks.create
func ensureSinkCreated(ctx context.Context, cli *logadmin.Client, topicResName *v1alpha1.GCloudResourceName) (*logadmin.Sink, error) {
	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudAuditLogsSource)
	status := &src.Status

	sinkID := generateSinkID(src)

	sink, err := cli.Sink(ctx, sinkID)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Cloud Audit Logs API: "+toErrMsg(err))
		return nil, controller.NewPermanentError(failCreatingAuditLogsSink(sinkID, err))
	case isNotFound(err):
		opts := []FilterOption{}
		if src.Spec.ResourceName != nil {
			opts = append(opts, WithResourceName(*src.Spec.ResourceName))
		}
		filterBuilder := NewFilterBuilder(src.Spec.ServiceName, src.Spec.MethodName, opts...)

		sink = &logadmin.Sink{
			ID:          sinkID,
			Destination: generateTopicResourceName(src, topicResName.Resource),
			Filter:      filterBuilder.GetFilter(),
		}
		sink, err = cli.CreateSinkOpt(ctx, sink, logadmin.SinkOptions{UniqueWriterIdentity: true})
		switch {
		case isDenied(err):
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Access denied to Cloud Audit Logs API: "+toErrMsg(err))
			return nil, controller.NewPermanentError(failCreatingAuditLogsSink(sinkID, err))
		case err != nil:
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Cannot create sink: "+toErrMsg(err))
			return nil, reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
				"Failed to create sink %q: %s", sinkID, toErrMsg(err))
		}
		event.Normal(ctx, ReasonSubscribed, "Created Audit Logs Sink %q", sink.ID)
	case err != nil:
		return nil, reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Failed to create sink %q: %s", sinkID, toErrMsg(err))

	}

	status.AuditLogsSink = &sink.ID

	return sink, err
}

// Ensures that the sink has been granted the pubsub.publisher role on the source topic.
// Required permissions:
// - pubsub.topics.getIamPolicy
// - pubsub.topics.setIamPolicy
// - pubsub.topics.publish
func ensureSinkIsPublisher(ctx context.Context, sink *logadmin.Sink, cli *pubsub.Client, topicResName *v1alpha1.GCloudResourceName) error {
	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudAuditLogsSource)
	status := &src.Status

	topicIam := cli.Topic(topicResName.Resource).IAM()
	topicPolicy, err := topicIam.Policy(ctx)
	if err != nil {
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Cannot retrieve topic policy: "+toErrMsg(err))
		return err
	}

	if !topicPolicy.HasRole(sink.WriterIdentity, publisherRole) {
		topicPolicy.Add(sink.WriterIdentity, publisherRole)
		if err = topicIam.SetPolicy(ctx, topicPolicy); err != nil {
			status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
				"Cannot set "+publisherRole+" topic policy: "+toErrMsg(err))
			return err
		}
		event.Normal(ctx, ReasonSubscribed, "Audit Logs Sink configured %q", sink.ID)
	}

	status.MarkSubscribed()
	return nil
}

// EnsureNoSink looks at status.AuditLogSink and if non-empty will delete it
// Required permissions:
// - logging.sinks.get
// - logging.sinks.delete
func EnsureNoSink(ctx context.Context, cli *logadmin.Client) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := commonv1alpha1.ReconcilableFromContext(ctx).(*v1alpha1.GoogleCloudAuditLogsSource)
	status := &src.Status

	sink := status.AuditLogsSink

	if sink == nil {
		return nil
	}

	err := cli.DeleteSink(ctx, *sink)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to Cloud Audit Log API. Ignoring: %s", toErrMsg(err))
		return nil
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed,
			fmt.Sprintf("Sink %q not found, skipping deletion", *sink))
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Failed to delete sink %q: %s", *sink, toErrMsg(err))
	}

	event.Normal(ctx, ReasonUnsubscribed, "Deleted Sink with ID "+*sink)

	return nil
}

// failCreatingAuditLogsSink returns a reconciler event which indicates
// that an Audit Log Sink could not be retrieved or created from the
// Google Cloud API.
func failCreatingAuditLogsSink(sink string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error creating Audit Logs Sink %q: %s", sink, toErrMsg(origErr))
}

// Generates the resource name for the topic used by an CloudAuditLogsSource.
func generateTopicResourceName(s *v1alpha1.GoogleCloudAuditLogsSource, topicID string) string {
	return fmt.Sprintf("pubsub.googleapis.com/projects/%s/topics/%s", *s.Spec.PubSub.Project, topicID)
}

// generateSinkID returns a deterministic resource ID for an Audit Logs router sink.
//
// The generated ID can contain up to 100 characters (lower-case alphanumeric
// characters, underscores, hyphens, and periods), which doesn't give us a lot
// of characters for indicating what component owns the logs router sink.
// Therefore, we compute the CRC32 checksum of the source's name/namespace (8
// characters) and make it part of the name.
func generateSinkID(s *v1alpha1.GoogleCloudAuditLogsSource) string {
	nsNameChecksum := crc32.ChecksumIEEE([]byte(s.Namespace + "/" + s.Name))
	return "io.triggermesh.googlecloudauditlogssources-" + strconv.FormatUint(uint64(nsNameChecksum), 10)
}
