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
	"sort"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *AzureEventGridSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("AzureEventGridSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (s *AzureEventGridSource) GetConditionSet() apis.ConditionSet {
	return azureEventGridSourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *AzureEventGridSource) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetSink implements Reconcilable.
func (s *AzureEventGridSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *AzureEventGridSource) GetStatusManager() *StatusManager {
	return &StatusManager{
		ConditionSet:      s.GetConditionSet(),
		EventSourceStatus: &s.Status.EventSourceStatus,
	}
}

// AsEventSource implements Reconcilable.
func (s *AzureEventGridSource) AsEventSource() string {
	return s.Spec.Scope.String()
}

// GetEventTypes returns the event types generated by the source.
func (s *AzureEventGridSource) GetEventTypes() []string {
	if len(s.Spec.EventTypes) == 0 {
		return nil
	}

	selectedTypes := make(map[string]struct{})
	for _, t := range s.Spec.EventTypes {
		if _, alreadySet := selectedTypes[t]; !alreadySet {
			selectedTypes[t] = struct{}{}
		}
	}

	eventTypes := make([]string, 0, len(selectedTypes))

	for t := range selectedTypes {
		eventTypes = append(eventTypes, t)
	}

	sort.Strings(eventTypes)

	return eventTypes
}

// Status conditions
const (
	// AzureEventGridConditionSubscribed has status True when an event subscription exists for the source.
	AzureEventGridConditionSubscribed apis.ConditionType = "Subscribed"
)

// azureEventGridSourceConditionSet is a set of conditions for
// AzureEventGridSource objects.
var azureEventGridSourceConditionSet = NewEventSourceConditionSet(
	AzureEventGridConditionSubscribed,
)

// MarkSubscribed sets the Subscribed condition to True.
func (s *AzureEventGridSourceStatus) MarkSubscribed() {
	azureEventGridSourceConditionSet.Manage(s).MarkTrue(AzureEventGridConditionSubscribed)
}

// MarkNotSubscribed sets the Subscribed condition to False with the given
// reason and message.
func (s *AzureEventGridSourceStatus) MarkNotSubscribed(reason, msg string) {
	azureEventGridSourceConditionSet.Manage(s).MarkFalse(AzureEventGridConditionSubscribed, reason, msg)
}
