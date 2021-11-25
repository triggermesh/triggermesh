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
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/common/resource"
)

// EventFlowComponent is implemented by all event flow components.
type EventFlowComponent interface {
	metav1.Object
	runtime.Object
	// OwnerRefable is used to construct a generic reconciler for each
	// source type, and convert source objects to owner references.
	kmeta.OwnerRefable
	// KRShaped is used by generated reconcilers to perform pre and
	// post-reconcile status updates.
	duckv1.KRShaped
	// GetStatusManager returns a manager for the component's status.
	GetStatusManager() *EventFlowStatusManager
	// AsEventSource returns a unique reference to the source suitable for
	// use as a CloudEvent source attribute.
	AsEventSource() string
}

// multiTenant is implemented by all multi-tenant source types.
type multiTenant interface {
	IsMultiTenant() bool
}

// IsMultiTenant returns whether the given component type is multi-tenant.
func IsMultiTenant(src EventFlowComponent) bool {
	mt, ok := src.(multiTenant)
	return ok && mt.IsMultiTenant()
}

// serviceAccountProvider is implemented by component types which are able to
// influence the shape of the ServiceAccount used by their own receive adapter.
type serviceAccountProvider interface {
	WantsOwnServiceAccount() bool
	ServiceAccountOptions() []resource.ServiceAccountOption
}

// WantsOwnServiceAccount returns whether the given component instance should have
// a dedicated ServiceAccount associated with its receive adapter.
func WantsOwnServiceAccount(src EventFlowComponent) bool {
	saProvider, ok := src.(serviceAccountProvider)
	return ok && saProvider.WantsOwnServiceAccount()
}

// ServiceAccountOptions returns functional options for mutating the
// ServiceAccount associated with a given source instance.
func ServiceAccountOptions(src EventFlowComponent) []resource.ServiceAccountOption {
	saProvider, ok := src.(serviceAccountProvider)
	if !ok {
		return nil
	}

	return saProvider.ServiceAccountOptions()
}

type eventFlowComponentKey struct{}

// WithEventFlowComponent returns a copy of the parent context in which the value
// associated with the component key is the given flow component.
func WithEventFlowComponent(ctx context.Context, s EventFlowComponent) context.Context {
	return context.WithValue(ctx, eventFlowComponentKey{}, s)
}

// FlowComponentFromContext returns the event flow component stored in the context.
func EventFlowComponentFromContext(ctx context.Context) EventFlowComponent {
	if s, ok := ctx.Value(eventFlowComponentKey{}).(EventFlowComponent); ok {
		return s
	}
	return nil
}
