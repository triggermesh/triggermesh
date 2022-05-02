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

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*TwilioSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TwilioSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (s *TwilioSource) GetConditionSet() pkgapis.ConditionSet {
	return v1alpha1.EventSenderConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *TwilioSource) GetStatus() *duckv1.Status {
	return &s.Status.Status
}

// GetSink implements EventSender.
func (s *TwilioSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *TwilioSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status,
	}
}

// AsEventSource implements EventSource.
func (s *TwilioSource) AsEventSource() string {
	return TwilioSourceName(s.Namespace, s.Name)
}

// TwilioSourceName returns a unique reference to the source suitable for use
// as as a CloudEvent source.
func TwilioSourceName(namespace, name string) string {
	return "io.triggermesh.twilio/" + namespace + "/" + name
}

// Supported event types.
const (
	TwilioSourceGenericEventType = "com.triggermesh.twilio.sms"
)

// GetEventTypes implements EventSource.
func (s *TwilioSource) GetEventTypes() []string {
	return []string{
		TwilioSourceGenericEventType,
	}
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *TwilioSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}
