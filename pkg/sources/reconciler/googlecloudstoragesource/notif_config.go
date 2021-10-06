/*
Copyright 2021 TriggerMesh Inc.

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

package googlecloudstoragesource

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"cloud.google.com/go/storage"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
)

// ensureNotificationConfig ensures the existence of a notification
// configuration targetting the given Pub/Sub topic on a Cloud Storage bucket.
// Required permissions:
// - storage.buckets.get
// - storage.buckets.update
func ensureNotificationConfig(ctx context.Context, cli *storage.Client,
	topicResName *v1alpha1.GCloudResourceName) error {

	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx).(*v1alpha1.GoogleCloudStorageSource)
	status := &src.Status

	desiredNotif := &storage.Notification{
		EventTypes:     src.Spec.EventTypes,
		PayloadFormat:  storage.JSONPayload,
		TopicProjectID: topicResName.Project,
		TopicID:        topicResName.Resource,
	}

	bh := cli.Bucket(src.Spec.Bucket)

	currentNotif, err := getOrCreateNotificationConfig(ctx, bh, desiredNotif)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Cloud Storage API: "+toErrMsg(err))
		return controller.NewPermanentError(failObtainNotificationConfigEvent(src.Spec.Bucket, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Cannot obtain notification configuration: "+toErrMsg(err))
		// wrap any other error to fail the reconciliation
		return fmt.Errorf("%w", failObtainNotificationConfigEvent(src.Spec.Bucket, err))
	}

	if notifID := status.NotificationID; notifID == nil {
		event.Normal(ctx, ReasonSubscribed, "Added notification configuration with ID "+currentNotif.ID)
	}

	notifID := currentNotif.ID

	currentNotif, err = syncNotificationConfig(ctx, bh, desiredNotif, currentNotif)
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Cloud Storage API: "+toErrMsg(err))
		return controller.NewPermanentError(failUpdateNotificationConfigEvent(src.Spec.Bucket, err))
	case err != nil:
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Cannot update notification configuration: "+toErrMsg(err))
		// wrap any other error to fail the reconciliation
		return fmt.Errorf("%w", failUpdateNotificationConfigEvent(src.Spec.Bucket, err))
	}

	if currentNotif.ID != notifID {
		event.Normal(ctx, ReasonSubscribed, "Re-created notification configuration with ID %s due to "+
			"changes in the source spec. New ID is %s", notifID, currentNotif.ID)
	}

	status.NotificationID = &currentNotif.ID
	status.MarkSubscribed()

	return nil
}

// ensureNoNotificationConfig ensures that the notification
// configuration is deleted.
// Required permissions:
// - storage.buckets.update
func ensureNoNotificationConfig(ctx context.Context, cli *storage.Client) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx).(*v1alpha1.GoogleCloudStorageSource)
	status := &src.Status

	notifID := status.NotificationID
	if notifID == nil {
		// notification configuration was possibly never created
		return nil
	}

	bh := cli.Bucket(src.Spec.Bucket)
	err := bh.DeleteNotification(ctx, *notifID)
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to Cloud Storage API. Ignoring: %s", toErrMsg(err))
		return nil
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed,
			fmt.Sprintf("Notification configuration %q not found, skipping deletion", *notifID))
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Failed to delete bucket notification configuration %q: %s", *notifID, toErrMsg(err))
	}

	event.Normal(ctx, ReasonUnsubscribed, "Deleted notification configuration with ID "+*notifID)

	return nil
}

// getOrCreateNotificationConfig returns a notification configuration matching
// the desired state, or creates one if it doesn't exist.
func getOrCreateNotificationConfig(ctx context.Context, bh *storage.BucketHandle,
	desired *storage.Notification) (*storage.Notification, error) {

	current, err := findNotificationConfig(ctx, bh, desired.TopicProjectID, desired.TopicID)
	if err != nil {
		return nil, fmt.Errorf("looking up notification configuration: %w", err)
	}

	if current == nil {
		current, err = bh.AddNotification(ctx, desired)
		if err != nil {
			return nil, fmt.Errorf("adding notification configuration: %w", err)
		}
	}

	return current, nil
}

// findNotificationConfig attempts to find a notification configuration which
// targets the given Pub/Sub topic.
func findNotificationConfig(ctx context.Context, bh *storage.BucketHandle,
	topicProject, topicID string) (*storage.Notification, error) {

	notifs, err := bh.Notifications(ctx)
	if err != nil {
		return nil, err
	}

	for _, n := range notifs {
		if n.TopicID == topicID && n.TopicProjectID == topicProject {
			return n, nil
		}
	}

	return nil, nil
}

// syncNotificationConfig ensures the current notification configuration has
// the desired state.
func syncNotificationConfig(ctx context.Context, bh *storage.BucketHandle,
	desired, current *storage.Notification) (*storage.Notification, error) {

	if equalNotificationConfig(desired, current) {
		return current, nil
	}

	err := bh.DeleteNotification(ctx, current.ID)
	switch {
	case isNotFound(err):
		// no-op, just re-create
	case err != nil:
		return nil, fmt.Errorf("deleting notification configuration for re-creation: %w", err)
	}

	current, err = bh.AddNotification(ctx, desired)
	if err != nil {
		return nil, fmt.Errorf("re-adding notification configuration: %w", err)
	}

	return current, nil
}

// equalNotificationConfig asserts the equality of two storage.Notification.
func equalNotificationConfig(desired, current *storage.Notification) bool {
	return cmp.Equal(desired, current,
		cmpopts.IgnoreFields(storage.Notification{}, "ID"),
		cmpopts.SortSlices(func(x, y string) bool { return x < y }),
	)
}

// failObtainNotificationConfigEvent returns a reconciler event which indicates
// that a notification configuration could not be retrieved or created from the
// Google Cloud API.
func failObtainNotificationConfigEvent(bucket string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error obtaining notification configuration for bucket %q: %s", bucket, toErrMsg(origErr))
}

// failUpdateNotificationConfigEvent returns a reconciler event which indicates
// that a notification configuration could not be deleted or re-added from the
// Google Cloud API.
func failUpdateNotificationConfigEvent(bucket string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error updating notification configuration for bucket %q: %s", bucket, toErrMsg(origErr))
}
