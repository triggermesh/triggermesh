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

package filter

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/reconciler"

	"github.com/triggermesh/triggermesh/pkg/apis/routing/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/routing/v1alpha1/filter"
)

// Reconciler implements controller.Reconciler for the event source type.
type Reconciler struct {
	adapter MTAdapter
}

// Check the interfaces Reconciler should implement.
var (
	_ reconcilerv1alpha1.Interface         = (*Reconciler)(nil)
	_ reconcilerv1alpha1.ReadOnlyInterface = (*Reconciler)(nil)
	_ reconcilerv1alpha1.ReadOnlyFinalizer = (*Reconciler)(nil)
)

// ReconcileKind implements reconcilerv1alpha1.Interface.
func (r *Reconciler) ReconcileKind(ctx context.Context, f *v1alpha1.Filter) reconciler.Event {
	return r.reconcile(ctx, f)
}

// ObserveKind implements reconcilerv1alpha1.ReadOnlyInterface.
func (r *Reconciler) ObserveKind(ctx context.Context, f *v1alpha1.Filter) reconciler.Event {
	return r.reconcile(ctx, f)
}

func (r *Reconciler) reconcile(ctx context.Context, f *v1alpha1.Filter) error {
	if f.Status.SinkURI == nil {
		// Mark that error as permanent so we don't retry until the
		// source's status has been updated, which automatically
		// triggers a new reconciliation.
		return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonSourceNotReady,
			"Event sink URL wasn't resolved yet. Skipping adapter configuration"))
	}

	if err := r.adapter.RegisterHandlerFor(ctx, f); err != nil {
		return fmt.Errorf("registering HTTP handler: %w", err)
	}

	return nil
}

// ObserveFinalizeKind implements reconcilerv1alpha1.ReadOnlyFinalizer.
func (r *Reconciler) ObserveFinalizeKind(ctx context.Context, f *v1alpha1.Filter) reconciler.Event {
	return r.finalize(ctx, f)
}

func (r *Reconciler) finalize(ctx context.Context, f *v1alpha1.Filter) error {
	if err := r.adapter.DeregisterHandlerFor(ctx, f); err != nil {
		return fmt.Errorf("deregistering HTTP handler: %w", err)
	}

	return reconciler.NewEvent(corev1.EventTypeNormal, ReasonHandlerDeregistered,
		"HTTP handler deregistered")
}
