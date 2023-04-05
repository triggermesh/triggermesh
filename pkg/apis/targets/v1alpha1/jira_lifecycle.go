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

// Managed event types
const (
	EventTypeJiraIssueCreate = "io.triggermesh.jira.issue.create"
	EventTypeJiraIssueGet    = "io.triggermesh.jira.issue.get"
	EventTypeJiraCustom      = "io.triggermesh.jira.custom"

	EventTypeJiraIssue          = "io.triggermesh.jira.issue"
	EventTypeJiraCustomResponse = "io.triggermesh.jira.custom.response"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*JiraTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("JiraTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*JiraTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *JiraTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *JiraTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*JiraTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeJiraIssueCreate,
		EventTypeJiraIssueGet,
		EventTypeJiraCustom,
	}
}

// GetEventTypes implements EventSource.
func (*JiraTarget) GetEventTypes() []string {
	return []string{
		EventTypeJiraIssue,
		EventTypeJiraCustomResponse,
	}
}

// AsEventSource implements EventSource.
func (t *JiraTarget) AsEventSource() string {
	kind := strings.ToLower(t.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + t.Namespace + "." + t.Name
}

// GetAdapterOverrides implements AdapterConfigurable.
func (t *JiraTarget) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return t.Spec.AdapterOverrides
}

// SetDefaults implements apis.Defaultable
func (t *JiraTarget) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (t *JiraTarget) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
