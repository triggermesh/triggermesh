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
)

// Accepted event types
const (
	// EventTypeAWSS3Put represents a task to put an object in S3.
	EventTypeAWSS3Put = "io.triggermesh.awss3.object.put"
)

// Returned event types
const (
	// EventTypeAWSS3Result contains the result of the processing of an S3 event.
	EventTypeAWSS3Result = "io.triggermesh.targets.aws.s3.result"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*AWSS3Target) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AWSS3Target")
}

// GetConditionSet implements duckv1.KRShaped.
func (*AWSS3Target) GetConditionSet() apis.ConditionSet {
	return targetConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *AWSS3Target) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *AWSS3Target) GetStatusManager() *StatusManager {
	return &StatusManager{
		ConditionSet: t.GetConditionSet(),
		TargetStatus: &t.Status,
	}
}

// AcceptedEventTypes implements IntegrationTarget.
func (*AWSS3Target) AcceptedEventTypes() []string {
	return []string{
		EventTypeAWSS3Put,
		EventTypeWildcard,
	}
}

// GetEventTypes implements EventSource.
func (*AWSS3Target) GetEventTypes() []string {
	return []string{
		EventTypeAWSS3Result,
	}
}

// AsEventSource implements EventSource.
func (t *AWSS3Target) AsEventSource() string {
	return t.Spec.ARN
}
