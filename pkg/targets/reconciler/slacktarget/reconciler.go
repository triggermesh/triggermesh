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

package slacktarget

import (
	"context"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"

	slack "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	reconcilers "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/slacktarget"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
)

// reconciler reconciles the target adapter object
type reconciler struct {
	TargetAdapterImage string `envconfig:"SLACK_ADAPTER_IMAGE" default:"gcr.io/triggermesh/slack-target-adapter"`

	ksvcr libreconciler.KServiceReconciler

	// Configuration accessor for logging/metrics/tracing
	configs source.ConfigAccessor
}

// Check that our Reconciler implements Interface
var _ reconcilers.Interface = (*reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *reconciler) ReconcileKind(ctx context.Context, trg *slack.SlackTarget) pkgreconciler.Event {
	trg.Status.InitializeConditions()
	trg.Status.ObservedGeneration = trg.Generation
	trg.Status.AcceptedEventTypes = trg.AcceptedEventTypes()
	// NOTE(antoineco): such events aren't currently returned by the adapter.
	trg.Status.ResponseAttributes = libreconciler.CeResponseAttributes(trg)

	adapter, event := r.ksvcr.ReconcileKService(ctx, trg, MakeTargetAdapterKService(&TargetAdapterArgs{
		Image:   r.TargetAdapterImage,
		Configs: r.configs,
		Target:  trg,
	}))

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
