/*
Copyright 2023 TriggerMesh Inc.

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

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *SolaceSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("SolaceSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (s *SolaceSource) GetConditionSet() apis.ConditionSet {
	return v1alpha1.EventSenderConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *SolaceSource) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetSink implements EventSender.
func (s *SolaceSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *SolaceSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status,
	}
}

// Managed event types
const (
	SolaceSourceEventType = "io.triggermesh.solace.event"
)

// GetEventTypes implements EventSource.
func (*SolaceSource) GetEventTypes() []string {
	return []string{
		SolaceSourceEventType,
	}
}

// AsEventSource implements EventSource.
func (s *SolaceSource) AsEventSource() string {
	return s.Spec.QueueName
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *SolaceSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// SetDefaults implements apis.Defaultable
func (s *SolaceSource) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (s *SolaceSource) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
