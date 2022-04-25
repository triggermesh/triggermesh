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

package azureactivitylogssource

import (
	"context"
	"fmt"

	"knative.dev/pkg/reconciler"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/azureactivitylogssource"
	listersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/insights"
)

// Reconciler implements controller.Reconciler for the event target type.
type Reconciler struct {
	// Getter than can obtain clients for interacting with Azure APIs
	cg insights.ClientGetter

	// Event Hubs adapter
	base       common.GenericDeploymentReconciler[*v1alpha1.AzureActivityLogsSource, listersv1alpha1.AzureActivityLogsSourceNamespaceLister]
	adapterCfg *adapterConfig
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// Check that our Reconciler implements Finalizer
var _ reconcilerv1alpha1.Finalizer = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, o *v1alpha1.AzureActivityLogsSource) reconciler.Event {
	// inject source into context for usage in reconciliation logic
	ctx = commonv1alpha1.WithReconcilable(ctx, o)

	if err := r.ensureDiagnosticSettings(ctx); err != nil {
		return fmt.Errorf("failed to ensure Diagnostic Settings: %w", err)
	}

	return r.base.ReconcileAdapter(ctx, r)
}

// FinalizeKind is called when the resource is deleted.
func (r *Reconciler) FinalizeKind(ctx context.Context, o *v1alpha1.AzureActivityLogsSource) reconciler.Event {
	// inject source into context for usage in finalization logic
	ctx = commonv1alpha1.WithReconcilable(ctx, o)

	// The finalizer blocks the deletion of the source object until the
	// deletion of the Diagnostic Settings succeeds to ensure that we don't
	// leave any dangling resource behind us.
	return r.ensureNoDiagnosticSettings(ctx)
}
