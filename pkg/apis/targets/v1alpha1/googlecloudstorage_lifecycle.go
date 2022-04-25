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

// Managed event types
const (
	EventTypeGoogleCloudStorageObjectInsert = "com.google.cloud.storage.object.insert"

	EventTypeGoogleCloudStorageResponse = "com.google.cloud.storage.object.insert.response"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*GoogleCloudStorageTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GoogleCloudStorageTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*GoogleCloudStorageTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *GoogleCloudStorageTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *GoogleCloudStorageTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*GoogleCloudStorageTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeGoogleCloudStorageObjectInsert,
		EventTypeWildcard,
	}
}

// GetEventTypes implements EventSource.
func (*GoogleCloudStorageTarget) GetEventTypes() []string {
	return []string{
		EventTypeGoogleCloudStorageResponse,
	}
}

// AsEventSource implements EventSource.
func (t *GoogleCloudStorageTarget) AsEventSource() string {
	kind := strings.ToLower(t.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + t.Namespace + "." + t.Name
}

// GetAdapterOverrides implements AdapterConfigurable.
func (t *GoogleCloudStorageTarget) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return t.Spec.AdapterOverrides
}
