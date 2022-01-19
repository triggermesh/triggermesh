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

package xmltojsontransformation

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	pkgreconciler "knative.dev/pkg/reconciler"

	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/flow/v1alpha1/xmltojsontransformation"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
)

// Reconciler implements controller.Reconciler for the event target type.
type Reconciler struct {
	base common.GenericDeploymentReconciler

	// adapter properties
	adapterCfg *adapterConfig

	// Knative Service reconciler
	ksvcr libreconciler.KServiceReconciler
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, trg *v1alpha1.XMLToJSONTransformation) pkgreconciler.Event {
	sink, err := r.base.SinkResolver.URIFromDestinationV1(ctx, trg.Spec.Sink, trg)

	if err != nil {
		return pkgreconciler.NewEvent(corev1.EventTypeWarning, "SinkResolver", err.Error())
	}

	adapter, event := r.ksvcr.ReconcileKService(ctx, trg, makeTransformationAdapterKService(trg, r.adapterCfg, sink))

	trg.Status.PropagateKServiceAvailability(adapter)

	return event
}
