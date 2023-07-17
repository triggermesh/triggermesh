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

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

// Managed event types
const (
	GoogleCloudPubSubResponseEventType = "io.triggermesh.google.pubsub.response"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*GoogleCloudPubSubTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GoogleCloudPubSubTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*GoogleCloudPubSubTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *GoogleCloudPubSubTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *GoogleCloudPubSubTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*GoogleCloudPubSubTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeWildcard,
	}
}

// GetEventTypes implements EventSource.
func (*GoogleCloudPubSubTarget) GetEventTypes() []string {
	return []string{
		GoogleCloudPubSubResponseEventType,
	}
}

// AsEventSource implements EventSource.
func (t *GoogleCloudPubSubTarget) AsEventSource() string {
	return t.Spec.Topic.String()
}

// GetAdapterOverrides implements AdapterConfigurable.
func (t *GoogleCloudPubSubTarget) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return t.Spec.AdapterOverrides
}

// WantsOwnServiceAccount implements ServiceAccountProvider.
func (t *GoogleCloudPubSubTarget) WantsOwnServiceAccount() bool {
	return t.Spec.Auth != nil && t.Spec.Auth.WantsOwnServiceAccount()
}

// ServiceAccountOptions implements ServiceAccountProvider.
func (t *GoogleCloudPubSubTarget) ServiceAccountOptions() []resource.ServiceAccountOption {
	if t.Spec.Auth == nil {
		return []resource.ServiceAccountOption{}
	}
	return t.Spec.Auth.ServiceAccountOptions()
}

// SetDefaults implements apis.Defaultable
func (t *GoogleCloudPubSubTarget) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (t *GoogleCloudPubSubTarget) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
