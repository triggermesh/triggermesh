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

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *AWSSQSSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AWSSQSSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (s *AWSSQSSource) GetConditionSet() apis.ConditionSet {
	return v1alpha1.EventSenderConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *AWSSQSSource) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetSink implements EventSender.
func (s *AWSSQSSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *AWSSQSSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status,
	}
}

// Supported event types
const (
	AWSSQSGenericEventType = "message"
)

// GetEventTypes implements EventSource.
func (s *AWSSQSSource) GetEventTypes() []string {
	return []string{
		AWSEventType(s.Spec.ARN.Service, AWSSQSGenericEventType),
	}
}

// AsEventSource implements EventSource.
func (s *AWSSQSSource) AsEventSource() string {
	return s.Spec.ARN.String()
}

// WantsOwnServiceAccount implements ServiceAccountProvider.
func (s *AWSSQSSource) WantsOwnServiceAccount() bool {
	return s.Spec.Auth.WantsOwnServiceAccount()
}

// ServiceAccountOptions implements ServiceAccountProvider.
func (s *AWSSQSSource) ServiceAccountOptions() []resource.ServiceAccountOption {
	return s.Spec.Auth.ServiceAccountOptions()
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *AWSSQSSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// SetDefaults implements apis.Defaultable
func (s *AWSSQSSource) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (s *AWSSQSSource) Validate(ctx context.Context) *apis.FieldError {
	// Do not validate authentication object in case of resource deletion
	if s.DeletionTimestamp != nil {
		return nil
	}
	return s.Spec.Auth.Validate(ctx)
}
