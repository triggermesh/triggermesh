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
	// EventTypeSendGridEmailSend represents a task to send an email.
	EventTypeSendGridEmailSend = "io.triggermesh.sendgrid.email.send"
	// EventTypeSendGridEmailSendResponse represents a response from the API after sending an email
	EventTypeSendGridEmailSendResponse = "io.triggermesh.sendgrid.email.send.response"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*SendGridTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("SendGridTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*SendGridTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *SendGridTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *SendGridTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*SendGridTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeSendGridEmailSend,
		EventTypeWildcard,
	}
}

// GetEventTypes implements EventSource.
func (*SendGridTarget) GetEventTypes() []string {
	return []string{
		EventTypeResponse,
	}
}

// AsEventSource implements EventSource.
func (t *SendGridTarget) AsEventSource() string {
	kind := strings.ToLower(t.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + t.Namespace + "." + t.Name
}

// GetAdapterOverrides implements AdapterConfigurable.
func (t *SendGridTarget) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return t.Spec.AdapterOverrides
}

// SetDefaults implements apis.Defaultable
func (t *SendGridTarget) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (t *SendGridTarget) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
