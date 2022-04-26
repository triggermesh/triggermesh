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
	FilterGenericEventType = "io.triggermesh.routing.filter"
)

// GetGroupVersionKind implements kmeta.OwnerRefable
func (*Filter) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Filter")
}

// GetStatus implements duckv1.KRShaped.
func (f *Filter) GetStatus() *duckv1.Status {
	return &f.Status.Status
}

// GetConditionSet implements duckv1.KRShaped.
func (*Filter) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatusManager implements Reconcilable.
func (f *Filter) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: f.GetConditionSet(),
		Status:       &f.Status,
	}
}

// GetEventTypes implements EventSource.
func (*Filter) GetEventTypes() []string {
	return []string{
		FilterGenericEventType,
	}
}

// AsEventSource implements EventSource.
func (f *Filter) AsEventSource() string {
	return "filter/" + f.Name
}

// GetSink implements EventSender.
func (f *Filter) GetSink() *duckv1.Destination {
	return f.Spec.Sink
}

// IsMultiTenant implements MultiTenant.
func (*Filter) IsMultiTenant() bool {
	return true
}

// GetAdapterOverrides implements AdapterConfigurable.
func (f *Filter) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return f.Spec.AdapterOverrides
}
