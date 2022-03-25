/*
Copyright 2021 TriggerMesh Inc.

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
	return targetConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *AWSDynamoDBTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *AWSDynamoDBTarget) GetStatusManager() *StatusManager {
	return &StatusManager{
		ConditionSet: t.GetConditionSet(),
		TargetStatus: &t.Status,
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
