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
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// Managed event types
const (
	IBMMQSourceEventType = "io.triggermesh.ibm.mq.message"
)

// GetEventTypes implements EventSource.
func (*IBMMQSource) GetEventTypes() []string {
	return []string{
		IBMMQSourceEventType,
	}
}

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*IBMMQSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("IBMMQSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (*IBMMQSource) GetConditionSet() apis.ConditionSet {
	return v1alpha1.DefaultConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *IBMMQSource) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetSink implements EventSender.
func (s *IBMMQSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *IBMMQSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status,
	}
}

// AsEventSource implements EventSource.
func (s *IBMMQSource) AsEventSource() string {
	return fmt.Sprintf("%s/%s", s.Spec.ConnectionName, strings.ToLower(s.Spec.ChannelName))
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *IBMMQSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// SetDefaults implements apis.Defaultable
func (s *IBMMQSource) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (s *IBMMQSource) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
