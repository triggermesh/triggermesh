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

package xslt

import (
	"context"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"knative.dev/pkg/reconciler"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/flow/v1alpha1/xslttransform"
	listersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/flow/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/flow/reconciler/common"
)

// Reconciler implements controller.Reconciler for the event source type.
type Reconciler struct {
	base         common.GenericServiceReconciler
	secretClient func(namespace string) coreclientv1.SecretInterface
	adapterCfg   *adapterConfig

	srcLister func(namespace string) listersv1alpha1.XsltTransformNamespaceLister
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, obj *v1alpha1.XsltTransform) reconciler.Event {
	// inject source into context for usage in reconciliation logic
	ctx = v1alpha1.WithEventFlowComponent(ctx, obj)
	return r.base.ReconcileComponent(ctx, r)
}
