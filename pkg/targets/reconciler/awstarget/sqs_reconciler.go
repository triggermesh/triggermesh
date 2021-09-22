package awstarget

import (
	"context"

	"github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	pkgreconciler "knative.dev/pkg/reconciler"

	awsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	reconcilers "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/awssqstarget"
)

// Reconciler reconciles the target adapter object
type sqsReconciler struct {
	ksvcr reconciler.KServiceReconciler
	vg    reconciler.ValueGetter

	adapterCfg *adapterConfig
}

// Check that our Reconciler implements Interface
var _ reconcilers.Interface = (*sqsReconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *sqsReconciler) ReconcileKind(ctx context.Context, trg *awsv1alpha1.AWSSQSTarget) pkgreconciler.Event {
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

	adapter, event := r.ksvcr.ReconcileKService(ctx, trg, makeTargetSQSAdapterKService(trg, r.adapterCfg))

	if adapter != nil {
		trg.Status.PropagateKServiceAvailability(adapter)
	} else {
		trg.Status.MarkNoKService("ServicePending", event.Error())
	}

	return event
}
