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

package xslt

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/flow/v1alpha1/xslttransform"
)

// Reconciler implements controller.Reconciler for the event source type.
type Reconciler struct {
	adapter MTAdapter
}

// Check the interfaces Reconciler should implement.
var (
	_ reconcilerv1alpha1.Interface         = (*Reconciler)(nil)
	_ reconcilerv1alpha1.ReadOnlyInterface = (*Reconciler)(nil)
	_ reconciler.OnDeletionInterface       = (*Reconciler)(nil)
)

// ReconcileKind implements reconcilerv1alpha1.Interface.
func (r *Reconciler) ReconcileKind(ctx context.Context, src *v1alpha1.XsltTransform) reconciler.Event {
	return r.reconcile(ctx, src)
}

// ObserveKind implements reconcilerv1alpha1.ReadOnlyInterface.
func (r *Reconciler) ObserveKind(ctx context.Context, src *v1alpha1.XsltTransform) reconciler.Event {
	return r.reconcile(ctx, src)
}

func (r *Reconciler) reconcile(ctx context.Context, src *v1alpha1.XsltTransform) error {
	// if src.Status.SinkURI == nil {
	// 	// Mark that error as permanent so we don't retry until the
	// 	// source's status has been updated, which automatically
	// 	// triggers a new reconciliation.
	// 	return controller.NewPermanentError(reconciler.NewEvent(corev1.EventTypeWarning, ReasonSourceNotReady,
	// 		"Event sink URL wasn't resolved yet. Skipping adapter configuration"))
	// }

	if err := r.adapter.RegisterHandlerFor(ctx, src); err != nil {
		return fmt.Errorf("registering HTTP handler: %w", err)
	}

	return nil
}

// ObserveDeletion implements reconciler.OnDeletionInterface.
func (r *Reconciler) ObserveDeletion(ctx context.Context, key types.NamespacedName) error {
	src := &v1alpha1.XsltTransform{}
	src.SetName(key.Name)
	src.SetNamespace(key.Namespace)

	return r.finalize(ctx, src)
}

func (r *Reconciler) finalize(ctx context.Context, src *v1alpha1.XsltTransform) error {
	if err := r.adapter.DeregisterHandlerFor(ctx, src); err != nil {
		return fmt.Errorf("deregistering HTTP handler: %w", err)
	}

	logging.FromContext(ctx).Info("HTTP handler deregistered")

	return nil
}
