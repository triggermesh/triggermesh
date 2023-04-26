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

// EventTypeHTTPTargetRequest is the event type for HTTP target request.
const EventTypeHTTPTargetRequest = "io.triggermesh.http.request"

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*HTTPTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("HTTPTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*HTTPTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *HTTPTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *HTTPTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// GetAdapterOverrides implements AdapterConfigurable.
func (t *HTTPTarget) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return t.Spec.AdapterOverrides
}

// SetDefaults implements apis.Defaultable
func (t *HTTPTarget) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (t *HTTPTarget) Validate(ctx context.Context) *apis.FieldError {
	return nil
}

// AcceptedEventTypes implements IntegrationTarget.
func (*HTTPTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeHTTPTargetRequest,
		EventTypeWildcard,
	}
}

// GetEventTypes implements EventSource.
func (t *HTTPTarget) GetEventTypes() []string {
	eventType := EventTypeResponse
	if t.Spec.Response.EventType != "" {
		eventType = t.Spec.Response.EventType
	}
	return []string{
		eventType,
	}
}

// AsEventSource implements EventSource.
func (t *HTTPTarget) AsEventSource() string {
	eventSource := t.Spec.Response.EventSource
	if eventSource == "" {
		kind := strings.ToLower(t.GetGroupVersionKind().Kind)
		eventSource = "io.triggermesh." + kind + "." + t.Namespace + "." + t.Name
	}
	return eventSource
}
