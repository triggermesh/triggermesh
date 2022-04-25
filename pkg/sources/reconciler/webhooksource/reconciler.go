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

package webhooksource

import (
	"context"

	"knative.dev/pkg/reconciler"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/webhooksource"
	listersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/sources/v1alpha1"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
)

// Reconciler implements controller.Reconciler for the event source type.
type Reconciler struct {
	base       common.GenericServiceReconciler[*v1alpha1.WebhookSource, listersv1alpha1.WebhookSourceNamespaceLister]
	adapterCfg *adapterConfig
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, src *v1alpha1.WebhookSource) reconciler.Event {
	// inject source into context for usage in reconciliation logic
	ctx = commonv1alpha1.WithReconcilable(ctx, src)

	return r.base.ReconcileAdapter(ctx, r)
}
