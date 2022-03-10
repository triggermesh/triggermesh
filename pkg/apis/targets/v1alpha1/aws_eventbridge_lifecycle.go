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
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// Returned event types
const (
	// EventTypeAWSEventBridgeResult contains the result of the processing of an eventbridge event.
	EventTypeAWSEventBridgeResult = "io.triggermesh.targets.aws.eventbridge.result"
)

// AcceptedEventTypes implements IntegrationTarget.
func (*AWSEventBridgeTarget) AcceptedEventTypes() []string {
	return []string{
		EventTypeAWSEventBridgeResult,
		EventTypeWildcard,
	}
}

// GetEventTypes implements EventSource.
func (*AWSEventBridgeTarget) GetEventTypes() []string {
	return []string{
		EventTypeAWSEventBridgeResult,
	}
}

// AsEventSource implements targets.EventSource.
func (s *AWSEventBridgeTarget) AsEventSource() string {
	return s.Spec.ARN
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *AWSEventBridgeTarget) GetConditionSet() apis.ConditionSet {
	return AwsCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *AWSEventBridgeTarget) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
