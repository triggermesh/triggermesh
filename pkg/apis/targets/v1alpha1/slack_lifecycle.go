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

package v1alpha1

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// Accepted event types
const (
	// EventTypeSlackPostMessage represents Post message Slack API request.
	EventTypeSlackPostMessage = "com.slack.webapi.chat.postMessage"
	// EventTypeSlackScheduleMessage represents Schedule message Slack API request.
	EventTypeSlackScheduleMessage = "com.slack.webapi.chat.scheduleMessage"
	// EventTypeSlackUpdateMessage represents Update message Slack API request.
	EventTypeSlackUpdateMessage = "com.slack.webapi.chat.update"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*SlackTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("SlackTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*SlackTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *SlackTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *SlackTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*SlackTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeSlackPostMessage,
		EventTypeSlackScheduleMessage,
		EventTypeSlackUpdateMessage,
	}
}

// GetEventTypes implements EventSource.
func (*SlackTarget) GetEventTypes() []string {
	return []string{
		EventTypeResponse,
	}
}

// AsEventSource implements EventSource.
func (t *SlackTarget) AsEventSource() string {
	kind := strings.ToLower(t.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + t.Namespace + "." + t.Name
}

// GetAdapterOverrides implements AdapterConfigurable.
func (t *SlackTarget) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return t.Spec.AdapterOverrides
}

// SetDefaults implements apis.Defaultable
func (t *SlackTarget) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (t *SlackTarget) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
