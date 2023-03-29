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
	"github.com/triggermesh/triggermesh/pkg/reconciler/resource"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*GoogleCloudBillingSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GoogleCloudBillingSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (*GoogleCloudBillingSource) GetConditionSet() apis.ConditionSet {
	return GoogleCloudBillingSourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *GoogleCloudBillingSource) GetStatus() *duckv1.Status {
	return &s.Status.Status.Status
}

// GetSink implements EventSender.
func (s *GoogleCloudBillingSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *GoogleCloudBillingSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status.Status,
	}
}

// AsEventSource implements EventSource.
func (s *GoogleCloudBillingSource) AsEventSource() string {
	return s.Spec.BudgetID
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *GoogleCloudBillingSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// WantsOwnServiceAccount implements ServiceAccountProvider.
func (s *GoogleCloudBillingSource) WantsOwnServiceAccount() bool {
	return s.Spec.Auth != nil && s.Spec.Auth.GCPServiceAccount != nil
}

// ServiceAccountOptions implements ServiceAccountProvider.
func (s *GoogleCloudBillingSource) ServiceAccountOptions() []resource.ServiceAccountOption {
	saOpts := []resource.ServiceAccountOption{}
	if s.Spec.Auth == nil {
		return saOpts
	}
	if gcpSA := s.Spec.Auth.GCPServiceAccount; gcpSA != nil {
		saOpts = append(saOpts, v1alpha1.GcpServiceAccountAnnotation(*gcpSA))
	}
	if k8sSA := s.Spec.Auth.KubernetesServiceAccount; k8sSA != nil {
		saOpts = append(saOpts, resource.SetServiceAccountName(*k8sSA))
	}
	return saOpts
}

// Supported event types
const (
	GoogleCloudBillingGenericEventType = "com.google.cloud.billing.notification"
)

// GetEventTypes returns the event types generated by the source.
func (*GoogleCloudBillingSource) GetEventTypes() []string {
	return []string{
		GoogleCloudBillingGenericEventType,
	}
}

// Status conditions
const (
	// GoogleCloudBillingConditionSubscribed has status True when the source has subscribed to a topic.
	GoogleCloudBillingConditionSubscribed apis.ConditionType = "Subscribed"
)

// GoogleCloudBillingSourceConditionSet is a set of conditions for
// GoogleCloudBillingSource objects.
var GoogleCloudBillingSourceConditionSet = v1alpha1.NewConditionSet(
	GoogleCloudBillingConditionSubscribed,
)

// MarkSubscribed sets the Subscribed condition to True.
func (s *GoogleCloudBillingSourceStatus) MarkSubscribed() {
	GoogleCloudBillingSourceConditionSet.Manage(s).MarkTrue(GoogleCloudBillingConditionSubscribed)
}

// MarkNotSubscribed sets the Subscribed condition to False with the given
// reason and message.
func (s *GoogleCloudBillingSourceStatus) MarkNotSubscribed(reason, msg string) {
	GoogleCloudBillingSourceConditionSet.Manage(s).MarkFalse(GoogleCloudBillingConditionSubscribed, reason, msg)
}
