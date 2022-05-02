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
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// Accepted event types
const (
	// EventTypeHasuraQuery represents a map of GraphQL query variables (name/value pairs).
	EventTypeHasuraQuery = "io.triggermesh.graphql.query"
	// EventTypeHasuraQueryRaw represents a raw GraphQL query.
	EventTypeHasuraQueryRaw = "io.triggermesh.graphql.query.raw"
)

// Returned event types
const (
	// EventTypeHasuraResult contains the result of the processing of a Hasura event.
	EventTypeHasuraResult = "org.graphql.query.result"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*HasuraTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("HasuraTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*HasuraTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *HasuraTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *HasuraTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*HasuraTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeHasuraQuery,
		EventTypeHasuraQueryRaw,
	}
}

// GetEventTypes implements EventSource.
func (*HasuraTarget) GetEventTypes() []string {
	return []string{
		EventTypeHasuraResult,
	}
}

// AsEventSource implements EventSource.
func (t *HasuraTarget) AsEventSource() string {
	kind := strings.ToLower(t.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + t.Namespace + "." + t.Name
}

// GetAdapterOverrides implements AdapterConfigurable.
func (t *HasuraTarget) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return t.Spec.AdapterOverrides
}
