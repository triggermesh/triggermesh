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

package googlecloudiotsource

import (
	"context"
	"fmt"

	gcloudiot "google.golang.org/api/cloudiot/v1"

	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/event"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common/skip"
)

// Ensures that the IoT Registry has the topic associated.
// Required permissions:
// - cloudiot.registries.update
func ensureTopicAssociated(ctx context.Context, cli *gcloudiot.Service, topicResName *v1alpha1.GCloudResourceName) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx).(*v1alpha1.GoogleCloudIoTSource)
	status := &src.Status

	registryName := src.Spec.Registry.String()

	updateRegistryRequest := &gcloudiot.DeviceRegistry{
		Name: registryName,
		StateNotificationConfig: &gcloudiot.StateNotificationConfig{
			PubsubTopicName: topicResName.String(),
		},
		EventNotificationConfigs: []*gcloudiot.EventNotificationConfig{
			{
				PubsubTopicName: topicResName.String(),
			},
		},
	}

	patchRegistry := cli.Projects.Locations.Registries.Patch(registryName, updateRegistryRequest)
	patchRegistry.UpdateMask("event_notification_configs,state_notification_config.pubsub_topic_name")

	_, err := patchRegistry.Do()
	switch {
	case isDenied(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"Access denied to Cloud Source IoT API: "+toErrMsg(err))
		return controller.NewPermanentError(failCreatingRegistry(registryName, err))
	case isNotFound(err):
		status.MarkNotSubscribed(v1alpha1.GCloudReasonAPIError,
			"IoT Registry not found: "+toErrMsg(err))
		return controller.NewPermanentError(failCreatingRegistry(registryName, err))
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
			"Failed to create notification for IoT Registry %q: %s", registryName, toErrMsg(err))
	}

	event.Normal(ctx, ReasonSubscribed, "Created notification for IoT Registry %q", registryName)
	status.MarkSubscribed()

	return err
}

// ensureNoTopicAssociated ensure that the IoT Registry has the topic disassociated.
// Required permissions:
// - cloudiot.registries.update
func (r *Reconciler) ensureNoTopicAssociated(ctx context.Context, cli *gcloudiot.Service) error {
	if skip.Skip(ctx) {
		return nil
	}

	src := v1alpha1.SourceFromContext(ctx).(*v1alpha1.GoogleCloudIoTSource)

	registryName := src.Spec.Registry.String()

	updateRegistryRequest := &gcloudiot.DeviceRegistry{
		Name:                     registryName,
		StateNotificationConfig:  &gcloudiot.StateNotificationConfig{},
		EventNotificationConfigs: []*gcloudiot.EventNotificationConfig{},
	}

	patchRegistry := cli.Projects.Locations.Registries.Patch(registryName, updateRegistryRequest)

	patchRegistry.UpdateMask("event_notification_configs,state_notification_config.pubsub_topic_name")

	_, err := patchRegistry.Do()
	switch {
	case isDenied(err):
		// it is unlikely that we recover from auth errors in the
		// finalizer, so we simply record a warning event and return
		event.Warn(ctx, ReasonFailedUnsubscribe,
			"Access denied to Cloud Source IoT API. Ignoring: %s", toErrMsg(err))
		return nil
	case isNotFound(err):
		event.Warn(ctx, ReasonUnsubscribed,
			fmt.Sprintf("IoT Registry %q not found, skipping deletion", registryName))
		return nil
	case err != nil:
		return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedUnsubscribe,
			"Cannot delete IoT Registry notification %q: %s", registryName, toErrMsg(err))
	}

	event.Normal(ctx, ReasonUnsubscribed, "Deleted notification for IoT Registry %q", registryName)

	return err
}

// failCreatingRegistry returns a reconciler event which indicates
// that a IoT Registry could not be retrieved or created from the
// Google Cloud API.
func failCreatingRegistry(registryName string, origErr error) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeWarning, ReasonFailedSubscribe,
		"Error creating Notification for IoT Registry %q: %s", registryName, toErrMsg(origErr))
}
