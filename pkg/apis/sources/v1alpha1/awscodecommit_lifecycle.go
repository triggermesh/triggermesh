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
func (s *AWSCodeCommitSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AWSCodeCommitSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (s *AWSCodeCommitSource) GetConditionSet() apis.ConditionSet {
	return v1alpha1.EventSenderConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *AWSCodeCommitSource) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetSink implements EventSender.
func (s *AWSCodeCommitSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *AWSCodeCommitSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status,
	}
}

// GetEventTypes implements EventSource.
func (s *AWSCodeCommitSource) GetEventTypes() []string {
	types := make([]string, len(s.Spec.EventTypes))

	for i, typ := range s.Spec.EventTypes {
		types[i] = AWSEventType(s.Spec.ARN.Service, typ)
	}

	return types
}

// AsEventSource implements EventSource.
func (s *AWSCodeCommitSource) AsEventSource() string {
	return s.Spec.ARN.String()
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *AWSCodeCommitSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// WantsOwnServiceAccount implements ServiceAccountProvider.
func (s *AWSCodeCommitSource) WantsOwnServiceAccount() bool {
	return s.Spec.Auth.EksIAMRole != nil
}

// ServiceAccountOptions implements ServiceAccountProvider.
func (s *AWSCodeCommitSource) ServiceAccountOptions() []resource.ServiceAccountOption {
	var saOpts []resource.ServiceAccountOption

	if iamRole := s.Spec.Auth.EksIAMRole; iamRole != nil {
		saOpts = append(saOpts, iamRoleAnnotation(*iamRole))
	}

	return saOpts
}
