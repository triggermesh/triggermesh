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
	EventTypeSalesforceAPICall         = "io.triggermesh.salesforce.apicall"
	EventTypeSalesforceAPICallResponse = "io.triggermesh.salesforce.apicall.response"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*SalesforceTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("SalesforceTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*SalesforceTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *SalesforceTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *SalesforceTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*SalesforceTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeSalesforceAPICall,
	}
}

// GetEventTypes implements EventSource.
func (*SalesforceTarget) GetEventTypes() []string {
	return []string{
		EventTypeSalesforceAPICallResponse,
	}
}

// AsEventSource implements EventSource.
func (t *SalesforceTarget) AsEventSource() string {
	kind := strings.ToLower(t.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + t.Namespace + "." + t.Name
}

// GetAdapterOverrides implements AdapterConfigurable.
func (t *SalesforceTarget) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return t.Spec.AdapterOverrides
}

// SetDefaults implements apis.Defaultable
func (t *SalesforceTarget) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (t *SalesforceTarget) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
