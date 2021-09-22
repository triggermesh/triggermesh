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

package googlesheettarget

import (
	"context"
	"fmt"

	reconciler2 "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"knative.dev/eventing/pkg/reconciler/source"
	pkgreconciler "knative.dev/pkg/reconciler"

	gsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	reconcilergsv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/googlesheettarget"
)

// reconciler reconciles the target adapter object
type reconciler struct {
	TargetAdapterImage string `envconfig:"GOOGLESHEET_ADAPTER_IMAGE" default:"gcr.io/triggermesh/googlesheet-target-adapter"`

	ksvcr reconciler2.KServiceReconciler
	vg    reconciler2.ValueGetter

	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Check that our Reconciler implements Interface
var _ reconcilergsv1alpha1.Interface = (*reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *reconciler) ReconcileKind(ctx context.Context, trg *gsv1alpha1.GoogleSheetTarget) pkgreconciler.Event {
	trg.Status.InitializeConditions()
	trg.Status.ObservedGeneration = trg.Generation
	// NOTE(antoineco): the adapter currently doesn't evaluate the attributes of incoming events.
	trg.Status.AcceptedEventTypes = trg.AcceptedEventTypes()
	// NOTE(antoineco): such events aren't currently returned by the adapter.
	trg.Status.ResponseAttributes = reconciler2.CeResponseAttributes(trg)

	_, err := r.vg.FromSecret(ctx, trg.Namespace, trg.Spec.GoogleServiceAccount.SecretKeyRef)
	if err != nil {
		trg.Status.MarkNoSecrets(err)
		return err
	}
	trg.Status.MarkSecrets()

	adapter, event := r.ksvcr.ReconcileKService(ctx, trg, MakeTargetAdapterKService(&TargetAdapterArgs{
		Image:   r.TargetAdapterImage,
		Configs: r.configs,
		Target:  trg,
	}))
	if err != nil {
		return fmt.Errorf("failed to synchronize adapter Service: %w", err)
	}
	trg.Status.PropagateAvailability(adapter)

	return event
}
