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
//nolint:stylecheck
const (
	// EventTypeUiPathStartJob represents job data to be initiated
	EventTypeUiPathStartJob = "io.triggermesh.uipath.job.start"
	// EventTypeUiPathQueuePost represents queue data to be posted to UiPath
	EventTypeUiPathQueuePost = "io.triggermesh.uipath.queue.post"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*UiPathTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("UiPathTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*UiPathTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *UiPathTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *UiPathTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*UiPathTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeUiPathStartJob,
		EventTypeUiPathQueuePost,
	}
}

// GetEventTypes implements EventSource.
func (*UiPathTarget) GetEventTypes() []string {
	return []string{
		EventTypeResponse,
	}
}

// AsEventSource implements EventSource.
func (t *UiPathTarget) AsEventSource() string {
	kind := strings.ToLower(t.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + t.Namespace + "." + t.Name
}
