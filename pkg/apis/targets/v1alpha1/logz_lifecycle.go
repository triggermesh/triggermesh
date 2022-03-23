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
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// Managed event types
const (
	EventTypeLogzShipResponse = "io.triggermesh.logz.ship.response"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*LogzTarget) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("LogzTarget")
}

// GetConditionSet implements duckv1.KRShaped.
func (*LogzTarget) GetConditionSet() apis.ConditionSet {
	return targetConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (t *LogzTarget) GetStatus() *duckv1.Status {
	return &t.Status.Status
}

// GetStatusManager implements Reconcilable.
func (t *LogzTarget) GetStatusManager() *StatusManager {
	return &StatusManager{
		ConditionSet: t.GetConditionSet(),
		TargetStatus: &t.Status,
	}
}

// GetEventTypes implements EventSource.
func (*LogzTarget) GetEventTypes() []string {
	return []string{
		EventTypeLogzShipResponse,
	}
}

// AsEventSource implements EventSource.
func (s *LogzTarget) AsEventSource() string {
	kind := strings.ToLower(s.GetGroupVersionKind().Kind)
	return "io.triggermesh." + kind + "." + s.Namespace + "." + s.Name
}
