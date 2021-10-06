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

package awskinesistarget

import (
	"context"

	pkgreconciler "knative.dev/pkg/reconciler"

	awsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	reconcilers "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/awskinesistarget"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
)

// kinesisReconciler reconciles the target adapter object
type kinesisReconciler struct {
	ksvcr libreconciler.KServiceReconciler
	vg    libreconciler.ValueGetter

	adapterCfg *adapterConfig
}

// Check that our Reconciler implements Interface
var _ reconcilers.Interface = (*kinesisReconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *kinesisReconciler) ReconcileKind(ctx context.Context, trg *awsv1alpha1.AWSKinesisTarget) pkgreconciler.Event {
	trg.Status.InitializeConditions()
	trg.Status.ObservedGeneration = trg.Generation

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

	adapter, event := r.ksvcr.ReconcileKService(ctx, trg, makeTargetKinesisAdapterKService(trg, r.adapterCfg))

	if adapter != nil {
		trg.Status.PropagateKServiceAvailability(adapter)
	} else {
		trg.Status.MarkNoKService("ServicePending", event.Error())
	}

	return event
}
