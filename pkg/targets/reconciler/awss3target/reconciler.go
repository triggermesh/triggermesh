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

package awss3target

import (
	"context"

	pkgreconciler "knative.dev/pkg/reconciler"

	awsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	reconcilers "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/awss3target"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
)

// Reconciler reconciles the target adapter object
type Reconciler struct {
	ksvcr libreconciler.KServiceReconciler
	vg    libreconciler.ValueGetter

	adapterCfg *adapterConfig
}

// Check that our Reconciler implements Interface
var _ reconcilers.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, trg *awsv1alpha1.AWSS3Target) pkgreconciler.Event {
	trg.Status.InitializeConditions()
	trg.Status.ObservedGeneration = trg.Generation
	trg.Status.AcceptedEventTypes = trg.AcceptedEventTypes()
	trg.Status.ResponseAttributes = libreconciler.CeResponseAttributes(trg)

	if trg.Spec.AWSApiKey.SecretKeyRef != nil {
		_, err := r.vg.FromSecret(ctx, trg.Namespace, trg.Spec.AWSApiKey.SecretKeyRef)
		if err != nil {
			trg.Status.MarkNoSecrets("AwsApiKeySecretNotFound", "%s", err)
			return err
		}
	}
	if trg.Spec.AWSApiSecret.SecretKeyRef != nil {
		_, err := r.vg.FromSecret(ctx, trg.Namespace, trg.Spec.AWSApiSecret.SecretKeyRef)
		if err != nil {
			trg.Status.MarkNoSecrets("AwsApiSecretNotFound", "%s", err)
			return err
		}
	}
	trg.Status.MarkSecrets()

	adapter, event := r.ksvcr.ReconcileKService(ctx, trg, makeTargetAdapterKService(trg, r.adapterCfg))

	trg.Status.PropagateKServiceAvailability(adapter)

	return event
}
