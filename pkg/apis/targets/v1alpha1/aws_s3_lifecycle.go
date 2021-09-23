/*
Copyright (c) 2021 TriggerMesh Inc.

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
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// Accepted event types
const (
	// EventTypeAWSS3Put represents a task to put an object in S3.
	EventTypeAWSS3Put = "io.triggermesh.awss3.object.put"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*AWSS3Target) AcceptedEventTypes() []string {
	return []string{
		EventTypeAWSS3Put,
		EventTypeWildcard,
	}
}

// Returned event types
const (
	// EventTypeAWSS3Result contains the result of the processing of an S3 event.
	EventTypeAWSS3Result = "io.triggermesh.targets.aws.s3.result"
)

// GetEventTypes implements EventSource.
func (*AWSS3Target) GetEventTypes() []string {
	return []string{
		EventTypeAWSS3Result,
	}
}

// AsEventSource implements targets.EventSource.
func (s *AWSS3Target) AsEventSource() string {
	return s.Spec.ARN
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *AWSS3Target) GetConditionSet() apis.ConditionSet {
	return AwsCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *AWSS3Target) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
