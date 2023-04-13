/*
Copyright 2023 TriggerMesh Inc.

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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// Managed event types
const (
	EventTypeMongoDBInsert  = "io.triggermesh.mongodb.insert"
	EventTypeMongoDBQueryKV = "io.triggermesh.mongodb.query.kv"
	EventTypeMongoDBUpdate  = "io.triggermesh.mongodb.update"

	EventTypeMongoDBStaticResponse = "io.triggermesh.mongodb.response"
	EventTypeMongoDBQueryResponse  = "io.triggermesh.mongodb.query.response"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*MongoDBTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("MongoDBTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*MongoDBTarget) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *MongoDBTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *MongoDBTarget) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: t.GetConditionSet(),
		Status:       &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*MongoDBTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeMongoDBInsert,
		EventTypeMongoDBQueryKV,
		EventTypeMongoDBUpdate,
	}
}

// GetEventTypes implements EventSource.
func (t *MongoDBTarget) GetEventTypes() []string {
	return []string{
		EventTypeMongoDBStaticResponse,
		EventTypeMongoDBQueryResponse,
	}
}

// AsEventSource implements EventSource.
func (t *MongoDBTarget) AsEventSource() string {
	return t.Spec.Database
}

// GetAdapterOverrides implements AdapterConfigurable.
func (t *MongoDBTarget) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return t.Spec.AdapterOverrides
}

// SetDefaults implements apis.Defaultable
func (t *MongoDBTarget) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (t *MongoDBTarget) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
