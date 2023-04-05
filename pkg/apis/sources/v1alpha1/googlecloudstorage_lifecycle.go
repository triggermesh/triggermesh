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
func (*GoogleCloudStorageSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GoogleCloudStorageSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (*GoogleCloudStorageSource) GetConditionSet() apis.ConditionSet {
	return googleCloudStorageSourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *GoogleCloudStorageSource) GetStatus() *duckv1.Status {
	return &s.Status.Status.Status
}

// GetSink implements EventSender.
func (s *GoogleCloudStorageSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *GoogleCloudStorageSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status.Status,
	}
}

// AsEventSource implements EventSource.
func (s *GoogleCloudStorageSource) AsEventSource() string {
	return "gs://" + s.Spec.Bucket
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *GoogleCloudStorageSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// WantsOwnServiceAccount implements ServiceAccountProvider.
func (s *GoogleCloudStorageSource) WantsOwnServiceAccount() bool {
	return s.Spec.Auth != nil && s.Spec.Auth.GCPServiceAccount != nil
}

// ServiceAccountOptions implements ServiceAccountProvider.
func (s *GoogleCloudStorageSource) ServiceAccountOptions() []resource.ServiceAccountOption {
	saOpts := []resource.ServiceAccountOption{}
	if s.Spec.Auth == nil {
		return saOpts
	}
	if gcpSA := s.Spec.Auth.GCPServiceAccount; gcpSA != nil {
		saOpts = append(saOpts, v1alpha1.GcpServiceAccountAnnotation(*gcpSA))
	}
	return saOpts
}

// Supported event types
const (
	GoogleCloudStorageGenericEventType = "com.google.cloud.storage.notification"
)

// GetEventTypes returns the event types generated by the source.
func (*GoogleCloudStorageSource) GetEventTypes() []string {
	return []string{
		GoogleCloudStorageGenericEventType,
	}
}

// Status conditions
const (
	// GoogleCloudStorageConditionSubscribed has status True when the source has subscribed to a topic.
	GoogleCloudStorageConditionSubscribed apis.ConditionType = "Subscribed"
)

// googleCloudStorageSourceConditionSet is a set of conditions for
// GoogleCloudStorageSource objects.
var googleCloudStorageSourceConditionSet = v1alpha1.NewConditionSet(
	GoogleCloudStorageConditionSubscribed,
)

// MarkSubscribed sets the Subscribed condition to True.
func (s *GoogleCloudStorageSourceStatus) MarkSubscribed() {
	googleCloudStorageSourceConditionSet.Manage(s).MarkTrue(GoogleCloudStorageConditionSubscribed)
}

// MarkNotSubscribed sets the Subscribed condition to False with the given
// reason and message.
func (s *GoogleCloudStorageSourceStatus) MarkNotSubscribed(reason, msg string) {
	googleCloudStorageSourceConditionSet.Manage(s).MarkFalse(GoogleCloudStorageConditionSubscribed, reason, msg)
}

// SetDefaults implements apis.Defaultable
func (s *GoogleCloudStorageSource) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (s *GoogleCloudStorageSource) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
