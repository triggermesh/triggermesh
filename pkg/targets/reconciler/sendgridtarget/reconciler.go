/*
Copyright (c) 2021 TriggerMesh Inc.

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

package sendgridtarget

import (
	"context"

	reconciler2 "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"go.uber.org/zap"

	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"

	sendgridv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	reconcilersendgrid "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/sendgridtarget"
)

// Reconciler reconciles the target adapter object
type reconciler struct {
	logger *zap.SugaredLogger
	ksvcr  reconciler2.KServiceReconciler
	vg     reconciler2.ValueGetter

	adapterCfg *adapterConfig
}

// Check that our Reconciler implements Interface
var _ reconcilersendgrid.Interface = (*reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *reconciler) ReconcileKind(ctx context.Context, trg *sendgridv1alpha1.SendGridTarget) pkgreconciler.Event {
	trg.Status.InitializeConditions()
	trg.Status.ObservedGeneration = trg.Generation
	// NOTE(antoineco): the adapter currently doesn't evaluate the attributes of incoming events.
	trg.Status.AcceptedEventTypes = trg.AcceptedEventTypes()
	// NOTE(antoineco): such events aren't currently returned by the adapter.
	trg.Status.ResponseAttributes = reconciler2.CeResponseAttributes(trg)

	_, err := r.vg.FromSecret(ctx, trg.Namespace, trg.Spec.APIKey.SecretKeyRef)
	if err != nil {
		trg.Status.MarkNoSecrets("APIKeyNotFound", "%s", err)
		return err
	}
	trg.Status.MarkSecrets()

	adapter, event := r.ksvcr.ReconcileKService(ctx, trg, makeTargetAdapterKService(trg, r.adapterCfg))

	if event != nil {
		logging.FromContext(ctx).Debugf("returning because an event was raised reconciling adapter KService")
		if adapter == nil {
			trg.Status.MarkNoKService("ServicePending", event.Error())
		} else {
			trg.Status.PropagateKServiceAvailability(adapter)
		}
		return event
	}

	trg.Status.PropagateKServiceAvailability(adapter)

	return event
}
