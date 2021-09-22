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

package oracletarget

import (
	"context"

	reconciler2 "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"go.uber.org/zap"

	oraclev1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"

	reconcilers "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/oracletarget"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// Reconciler reconciles the target adapter object
type reconciler struct {
	logger *zap.SugaredLogger
	ksvcr  reconciler2.KServiceReconciler
	vg     reconciler2.ValueGetter

	adapterCfg *adapterConfig
}

// Check that our Reconciler implements Interface
var _ reconcilers.Interface = (*reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *reconciler) ReconcileKind(ctx context.Context, trg *oraclev1alpha1.OracleTarget) pkgreconciler.Event {
	trg.Status.InitializeConditions()
	trg.Status.ObservedGeneration = trg.Generation

	if trg.Spec.OracleApiPrivateKey.SecretKeyRef != nil {
		_, err := r.vg.FromSecret(ctx, trg.Namespace, trg.Spec.OracleApiPrivateKey.SecretKeyRef)
		if err != nil {
			trg.Status.MarkNoSecrets("OracleApiPrivateKeySecretNotFound", "%s", err)
			return err
		}
	}
	if trg.Spec.OracleApiPrivateKeyPassphrase.SecretKeyRef != nil {
		_, err := r.vg.FromSecret(ctx, trg.Namespace, trg.Spec.OracleApiPrivateKeyPassphrase.SecretKeyRef)
		if err != nil {
			trg.Status.MarkNoSecrets("OracleApiPrivateKeyPassphraseNotFound", "%s", err)
			return err
		}
	}
	if trg.Spec.OracleApiPrivateKeyFingerprint.SecretKeyRef != nil {
		_, err := r.vg.FromSecret(ctx, trg.Namespace, trg.Spec.OracleApiPrivateKeyFingerprint.SecretKeyRef)
		if err != nil {
			trg.Status.MarkNoSecrets("OracleApiPrivateKeyFingerprintNotFound", "%s", err)
			return err
		}
	}
	trg.Status.MarkSecrets()

	adapter, event := r.ksvcr.ReconcileKService(ctx, trg, makeTargetAdapterKService(trg, r.adapterCfg))

	if adapter != nil {
		trg.Status.PropagateKServiceAvailability(adapter)
	} else {
		trg.Status.MarkNoKService("ServicePending", event.Error())
	}

	return event
}
