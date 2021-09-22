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

package tektontarget

import (
	"context"
	"fmt"

	reconciler2 "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
	"go.uber.org/zap"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacclientv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	rbaclistersv1 "k8s.io/client-go/listers/rbac/v1"
	pkgreconciler "knative.dev/pkg/reconciler"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	reconcilers "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/tektontarget"
	listersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/targets/v1alpha1"
)

// reconciler reconciles the target adapter object
type reconciler struct {
	logger *zap.SugaredLogger
	ksvcr  reconciler2.KServiceReconciler
	vg     reconciler2.ValueGetter

	adapterCfg *adapterConfig

	// API clients
	saClient func(namespace string) coreclientv1.ServiceAccountInterface
	rbClient func(namespace string) rbacclientv1.RoleBindingInterface
	// objects listers
	targetLister func(namespace string) listersv1alpha1.TektonTargetNamespaceLister
	saLister     func(namespace string) corelistersv1.ServiceAccountNamespaceLister
	rbLister     func(namespace string) rbaclistersv1.RoleBindingNamespaceLister
}

// Check that our Reconciler implements Interface
var _ reconcilers.Interface = (*reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *reconciler) ReconcileKind(ctx context.Context, trg *v1alpha1.TektonTarget) pkgreconciler.Event {
	trg.Status.InitializeConditions()
	trg.Status.ObservedGeneration = trg.Generation
	trg.Status.AcceptedEventTypes = trg.AcceptedEventTypes()
	// NOTE(antoineco): such events aren't currently returned by the adapter.
	trg.Status.ResponseAttributes = reconciler2.CeResponseAttributes(trg)

	if err := r.reconcileServiceAccounts(ctx, trg.Namespace); err != nil {
		return fmt.Errorf("reconciling adapter ServiceAccount: %w", err)
	}

	adapter, event := r.ksvcr.ReconcileKService(ctx, trg, makeTargetAdapterKService(trg, r.adapterCfg))

	if adapter != nil && adapter.Status.Address != nil {
		trg.Status.PropagateKServiceAvailability(adapter)
	} else {
		eventErr := ""
		if event != nil {
			eventErr = event.Error()
		}
		trg.Status.MarkNoKService("ServicePending", eventErr)
	}

	return event
}
