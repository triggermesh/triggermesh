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
func (*GoogleCloudSourceRepositoriesSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GoogleCloudSourceRepositoriesSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (*GoogleCloudSourceRepositoriesSource) GetConditionSet() apis.ConditionSet {
	return googleCloudSourceRepoSourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *GoogleCloudSourceRepositoriesSource) GetStatus() *duckv1.Status {
	return &s.Status.Status.Status
}

// GetSink implements EventSender.
func (s *GoogleCloudSourceRepositoriesSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *GoogleCloudSourceRepositoriesSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status.Status,
	}
}

// AsEventSource implements EventSource.
func (s *GoogleCloudSourceRepositoriesSource) AsEventSource() string {
	return s.Spec.Repository.String()
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *GoogleCloudSourceRepositoriesSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// WantsOwnServiceAccount implements ServiceAccountProvider.
func (s *GoogleCloudSourceRepositoriesSource) WantsOwnServiceAccount() bool {
	return s.Spec.GCPServiceAccount != nil
}

// ServiceAccountOptions implements ServiceAccountProvider.
func (s *GoogleCloudSourceRepositoriesSource) ServiceAccountOptions() []resource.ServiceAccountOption {
	var saOpts []resource.ServiceAccountOption

	if gcpSA := s.Spec.GCPServiceAccount; gcpSA != nil {
		saOpts = append(saOpts, v1alpha1.GcpServiceAccountAnnotation(*gcpSA))
	}
	return saOpts
}

// Supported event types
const (
	GoogleCloudSourceRepoGenericEventType = "com.google.cloud.sourcerepo.notification"
)

// GetEventTypes returns the event types generated by the source.
func (*GoogleCloudSourceRepositoriesSource) GetEventTypes() []string {
	return []string{
		GoogleCloudSourceRepoGenericEventType,
	}
}

// Status conditions
const (
	// GoogleCloudSourceRepoConditionSubscribed has status True when the source has subscribed to a topic.
	GoogleCloudSourceRepoConditionSubscribed apis.ConditionType = "Subscribed"
)

// googleCloudSourceRepoSourceConditionSet is a set of conditions for
// GoogleCloudSourceRepositoriesSource objects.
var googleCloudSourceRepoSourceConditionSet = v1alpha1.NewConditionSet(
	GoogleCloudSourceRepoConditionSubscribed,
)

// MarkSubscribed sets the Subscribed condition to True.
func (s *GoogleCloudSourceRepositoriesSourceStatus) MarkSubscribed() {
	googleCloudSourceRepoSourceConditionSet.Manage(s).MarkTrue(GoogleCloudSourceRepoConditionSubscribed)
}

// MarkNotSubscribed sets the Subscribed condition to False with the given
// reason and message.
func (s *GoogleCloudSourceRepositoriesSourceStatus) MarkNotSubscribed(reason, msg string) {
	googleCloudSourceRepoSourceConditionSet.Manage(s).MarkFalse(GoogleCloudSourceRepoConditionSubscribed, reason, msg)
}
