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

// Returned event types
const (
	// EventTypeAWSDynamoDBResult contains the result of the processing of an S3 event.
	EventTypeAWSDynamoDBResult = "io.triggermesh.targets.aws.dynamodb.result"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*AWSDynamoDBTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AWSDynamoDBTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*AWSDynamoDBTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *AWSDynamoDBTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *AWSDynamoDBTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// GetEventTypes implements EventSource.
func (*AWSDynamoDBTarget) GetEventTypes() []string {
	return []string{
		EventTypeAWSDynamoDBResult,
	}
}

// AsEventSource implements EventSource.
func (t *AWSDynamoDBTarget) AsEventSource() string {
	return t.Spec.ARN
}

// GetAdapterOverrides implements AdapterConfigurable.
func (t *AWSDynamoDBTarget) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return t.Spec.AdapterOverrides
}

// WantsOwnServiceAccount implements ServiceAccountProvider.
func (t *AWSDynamoDBTarget) WantsOwnServiceAccount() bool {
	return t.Spec.Auth.WantsOwnServiceAccount()
}

// ServiceAccountOptions implements ServiceAccountProvider.
func (t *AWSDynamoDBTarget) ServiceAccountOptions() []resource.ServiceAccountOption {
	return t.Spec.Auth.ServiceAccountOptions()
}

// SetDefaults implements apis.Defaultable
func (t *AWSDynamoDBTarget) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (t *AWSDynamoDBTarget) Validate(ctx context.Context) *apis.FieldError {
	// Do not validate authentication object in case of resource deletion
	if t.DeletionTimestamp != nil {
		return nil
	}
	return t.Spec.Auth.Validate(ctx)
}
