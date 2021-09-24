/*
Copyright 2021 Triggermesh Inc.

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
)

// GetGroupVersionKind implements kmeta.OwnerRefable
func (*Filter) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Filter")
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (f *Filter) GetConditionSet() apis.ConditionSet {
	return routerConditionSet
}

// Supported event types
const (
	FilterGenericEventType = "io.triggermesh.routing.filter"
)

// GetEventTypes implements Router.
func (*Filter) GetEventTypes() []string {
	return []string{
		FilterGenericEventType,
	}
}

// GetSink implements Router.
func (f *Filter) GetSink() *duckv1.Destination {
	return f.Spec.Sink
}

// GetStatusManager implements Router.
func (f *Filter) GetStatusManager() *RouterStatusManager {
	return &RouterStatusManager{
		ConditionSet: f.GetConditionSet(),
		RouterStatus: &f.Status,
	}
}

// IsMultiTenant implements MultiTenant.
func (*Filter) IsMultiTenant() bool {
	return true
}
