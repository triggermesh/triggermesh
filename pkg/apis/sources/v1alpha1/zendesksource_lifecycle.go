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

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
)

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*ZendeskSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ZendeskSource")
}

// GetConditionSet implements duckv1.KRShaped.
func (*ZendeskSource) GetConditionSet() apis.ConditionSet {
	return zendeskSourceConditionSet
}

// GetStatus implements duckv1.KRShaped.
func (s *ZendeskSource) GetStatus() *duckv1.Status {
	return &s.Status.Status.Status
}

// GetSink implements EventSender.
func (s *ZendeskSource) GetSink() *duckv1.Destination {
	return &s.Spec.Sink
}

// GetStatusManager implements Reconcilable.
func (s *ZendeskSource) GetStatusManager() *v1alpha1.StatusManager {
	return &v1alpha1.StatusManager{
		ConditionSet: s.GetConditionSet(),
		Status:       &s.Status.Status,
	}
}

// AsEventSource implements EventSource.
func (s *ZendeskSource) AsEventSource() string {
	return s.Spec.Subdomain + ".zendesk.com/" + s.Namespace + "/" + s.Name
}

// IsMultiTenant implements MultiTenant.
func (*ZendeskSource) IsMultiTenant() bool {
	return true
}

// GetAdapterOverrides implements AdapterConfigurable.
func (s *ZendeskSource) GetAdapterOverrides() *v1alpha1.AdapterOverrides {
	return s.Spec.AdapterOverrides
}

// Supported event types
const (
	// ZendeskTicketCreatedEventType is generated upon creation of a Ticket.
	ZendeskTicketCreatedEventType = "com.zendesk.ticket.created"
)

// GetEventTypes implements EventSource.
func (*ZendeskSource) GetEventTypes() []string {
	return []string{
		ZendeskTicketCreatedEventType,
	}
}

// Status conditions
const (
	// ZendeskConditionTargetSynced has status True when the Zendesk Target and Trigger have been synced.
	ZendeskConditionTargetSynced apis.ConditionType = "TargetSynced"
)

// Reasons for status conditions
const (
	// ZendeskReasonNoURL is set on a TargetSynced condition when the adapter URL is empty.
	ZendeskReasonNoURL = "MissingAdapterURL"
	// ZendeskReasonNoSecret is set on a TargetSynced condition when required secrets can't be obtained.
	ZendeskReasonNoSecret = "MissingSecret"
	// ZendeskReasonFailedSync is set on a TargetSynced condition when a CRUD API call returns an error.
	ZendeskReasonFailedSync = "FailedSync"
)

// zendeskSourceConditionSet is a set of status conditions for ZendeskSource
// objects.
var zendeskSourceConditionSet = v1alpha1.NewConditionSet(
	ZendeskConditionTargetSynced,
)

// MarkTargetSynced sets the TargetSynced condition to True.
func (s *ZendeskSourceStatus) MarkTargetSynced() {
	zendeskSourceConditionSet.Manage(s).MarkTrue(ZendeskConditionTargetSynced)
}

// MarkTargetNotSynced sets the TargetSynced condition to False with the given
// reason and associated message.
func (s *ZendeskSourceStatus) MarkTargetNotSynced(reason, msg string) {
	zendeskSourceConditionSet.Manage(s).MarkFalse(ZendeskConditionTargetSynced,
		ZendeskReasonFailedSync, msg)
}

// SetDefaults implements apis.Defaultable
func (s *ZendeskSource) SetDefaults(ctx context.Context) {
}

// Validate implements apis.Validatable
func (s *ZendeskSource) Validate(ctx context.Context) *apis.FieldError {
	return nil
}
