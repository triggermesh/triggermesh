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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/common/resource"
)

// Reconcilable is implemented by all TriggerMesh components.
type Reconcilable interface {
	metav1.Object
	runtime.Object
	// OwnerRefable is used to construct a generic reconciler for each
	// component type, and convert custom objects to owner references.
	kmeta.OwnerRefable
	// KRShaped is used by generated reconcilers to perform pre and
	// post-reconcile status updates.
	duckv1.KRShaped

	// GetStatusManager returns a manager for the component's status.
	GetStatusManager() *StatusManager
}

// EventSource is implemented by types that emit events, either by sending them
// to a sink or by replying to incoming event requests.
type EventSource interface {
	// GetEventTypes returns the event types generated by the component.
	GetEventTypes() []string
	// AsEventSource returns a unique reference to the component suitable
	// for use as a CloudEvent 'source' attribute.
	AsEventSource() string
}

// EventSender is implemented by types that send events to a sink.
type EventSender interface {
	// GetSink returns the component's event sink.
	GetSink() *duckv1.Destination
}

// EventReceiver is implemented by types that receive and process events.
type EventReceiver interface {
	// AcceptedEventTypes returns the event types accepted by the target.
	AcceptedEventTypes() []string
}

// multiTenant is implemented by all multi-tenant component types.
type multiTenant interface {
	IsMultiTenant() bool
}

// IsMultiTenant returns whether the given component type is multi-tenant.
func IsMultiTenant(r Reconcilable) bool {
	mt, ok := r.(multiTenant)
	return ok && mt.IsMultiTenant()
}

// serviceAccountProvider is implemented by types which are able to influence
// the shape of the ServiceAccount used by their own receive adapter.
type serviceAccountProvider interface {
	WantsOwnServiceAccount() bool
	ServiceAccountOptions() []resource.ServiceAccountOption
}

// WantsOwnServiceAccount returns whether the given component instance should
// have a dedicated ServiceAccount associated with its receive adapter.
func WantsOwnServiceAccount(r Reconcilable) bool {
	saProvider, ok := r.(serviceAccountProvider)
	return ok && saProvider.WantsOwnServiceAccount()
}

// ServiceAccountOptions returns functional options for mutating the
// ServiceAccount associated with a given component instance.
func ServiceAccountOptions(r Reconcilable) []resource.ServiceAccountOption {
	saProvider, ok := r.(serviceAccountProvider)
	if !ok {
		return nil
	}

	return saProvider.ServiceAccountOptions()
}

type reconcilableInstanceKey struct{}

// WithReconcilable returns a copy of the parent context in which the value
// associated with the reconcilableInstanceKey is the given component instance.
func WithReconcilable(ctx context.Context, r Reconcilable) context.Context {
	return context.WithValue(ctx, reconcilableInstanceKey{}, r)
}

// ReconcilableFromContext returns the component instance stored in the context.
func ReconcilableFromContext(ctx context.Context) Reconcilable {
	if r, ok := ctx.Value(reconcilableInstanceKey{}).(Reconcilable); ok {
		return r
	}
	return nil
}
