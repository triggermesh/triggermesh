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

package ibmmqtarget

import (
	"context"

	pkgreconciler "knative.dev/pkg/reconciler"

	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/ibmmqtarget"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
)

// Reconciler implements controller.Reconciler for the event target type.
type Reconciler struct {
	// adapter properties
	adapterCfg *adapterConfig

	// Knative Service reconciler
	ksvcr libreconciler.KServiceReconciler
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, trg *v1alpha1.IBMMQTarget) pkgreconciler.Event {
	trg.Status.InitializeConditions()
	trg.Status.ObservedGeneration = trg.Generation
	trg.Status.AcceptedEventTypes = trg.AcceptedEventTypes()
	trg.Status.ResponseAttributes = libreconciler.CeResponseAttributes(trg)

	adapter, event := r.ksvcr.ReconcileKService(ctx, trg, makeTargetAdapterKService(trg, r.adapterCfg))

	trg.Status.PropagateKServiceAvailability(adapter)

	return event
}
