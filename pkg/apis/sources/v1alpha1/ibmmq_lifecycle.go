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
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// Managed event types
const (
	IBMMQSourceEventType = "io.triggermesh.ibm.mq.message"
)

// GetEventTypes implements Reconcilable.
func (*IBMMQSource) GetEventTypes() []string {
	return []string{
		IBMMQSourceEventType,
	}
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (s *IBMMQSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("IBMMQSource")
}

// IBMMQSourceCondSet is the group of possible conditions
var IBMMQSourceCondSet = apis.NewLivingConditionSet(
	ConditionDeployed,
)

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (s *IBMMQSource) GetConditionSet() apis.ConditionSet {
	return IBMMQSourceCondSet
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (s *IBMMQSource) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetSink implements Reconcilable.
func (s *IBMMQSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *IBMMQSource) GetStatusManager() *StatusManager {
	return &StatusManager{
		ConditionSet:      s.GetConditionSet(),
		EventSourceStatus: &s.Status.EventSourceStatus,
	}
}

// AsEventSource implements Reconcilable.
func (s *IBMMQSource) AsEventSource() string {
	return fmt.Sprintf("%s/%s", s.Spec.ConnectionName, strings.ToLower(s.Spec.ChannelName))
}
