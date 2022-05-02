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
	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// Supported event types
const (
	SplitterGenericEventType = "io.triggermesh.routing.splitter"
)

// GetGroupVersionKind implements kmeta.OwnerRefable
func (*Splitter) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Splitter")
}

// GetConditionSet implements duckv1.KRShaped.
func (*Splitter) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *Splitter) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetStatusManager implements Reconcilable.
func (s *Splitter) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status,
	}
}

// GetEventTypes implements EventSource.
func (*Splitter) GetEventTypes() []string {
	return []string{
		SplitterGenericEventType,
	}
}

// AsEventSource implements EventSource.
func (s *Splitter) AsEventSource() string {
	return "splitter/" + s.Name
}

// GetSink implements EventSender.
func (s *Splitter) GetSink() *duckv1.Destination {
	return s.Spec.Sink
}

// IsMultiTenant implements MultiTenant.
func (*Splitter) IsMultiTenant() bool {
	return true
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *Splitter) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}
