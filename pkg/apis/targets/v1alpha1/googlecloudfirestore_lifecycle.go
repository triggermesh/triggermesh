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
)

// Managed event types
const (
	EventTypeGoogleCloudFirestoreWriteResponse = "io.triggermesh.google.firestore.write.response"
	EventTypeGoogleCloudFirestoreWrite         = "io.triggermesh.google.firestore.write"

	EventTypeGoogleCloudFirestoreQueryTablesResponse = "io.triggermesh.google.firestore.query.tables.response"
	EventTypeGoogleCloudFirestoreQueryTables         = "io.triggermesh.google.firestore.query.tables"

	EventTypeGoogleCloudFirestoreQueryTableResponse = "io.triggermesh.google.firestore.query.table.response"
	EventTypeGoogleCloudFirestoreQueryTable         = "io.triggermesh.google.firestore.query.table"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*GoogleCloudFirestoreTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("GoogleCloudFirestoreTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*GoogleCloudFirestoreTarget) GetConditionSet() apis.ConditionSet {
	return targetConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *GoogleCloudFirestoreTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *GoogleCloudFirestoreTarget) GetStatusManager() *StatusManager {
	return &StatusManager{
		ConditionSet: t.GetConditionSet(),
		TargetStatus: &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*GoogleCloudFirestoreTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeGoogleCloudFirestoreWrite,
		EventTypeGoogleCloudFirestoreQueryTables,
		EventTypeGoogleCloudFirestoreQueryTable,
	}
}

// GetEventTypes implements EventSource.
func (*GoogleCloudFirestoreTarget) GetEventTypes() []string {
	return []string{
		EventTypeGoogleCloudFirestoreWriteResponse,
		EventTypeGoogleCloudFirestoreWrite,
		EventTypeGoogleCloudFirestoreQueryTablesResponse,
		EventTypeGoogleCloudFirestoreQueryTables,
		EventTypeGoogleCloudFirestoreQueryTableResponse,
		EventTypeGoogleCloudFirestoreQueryTable,
	}
}

// AsEventSource implements EventSource.
func (t *GoogleCloudFirestoreTarget) AsEventSource() string {
	kind := strings.ToLower(t.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + t.Namespace + "." + t.Name
}
