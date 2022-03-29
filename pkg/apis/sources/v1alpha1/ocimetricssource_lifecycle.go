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

	pkgapis "knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*OCIMetricsSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("OCIMetricsSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (*OCIMetricsSource) GetConditionSet() pkgapis.ConditionSet {
	return eventSourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *OCIMetricsSource) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetSink implements Reconcilable.
func (s *OCIMetricsSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *OCIMetricsSource) GetStatusManager() *StatusManager {
	return &StatusManager{
		ConditionSet:      s.GetConditionSet(),
		EventSourceStatus: &s.Status,
	}
}

// AsEventSource implements Reconcilable.
func (s *OCIMetricsSource) AsEventSource() string {
	return OCIGenerateEventSource(s.Namespace, s.Name)
}

// OCIGenerateEventSource generate the event source name to be used in the adapter
func OCIGenerateEventSource(namespace, name string) string {
	return "ocimetrics/" + namespace + "/" + name
}

// Supported event types
const (
	OCIMetricsGenericEventType = "com.oracle.cloud.monitoring"
)

// GetEventTypes implements Reconcilable.
func (*OCIMetricsSource) GetEventTypes() []string {
	return []string{
		OCIMetricsGenericEventType,
	}
}
