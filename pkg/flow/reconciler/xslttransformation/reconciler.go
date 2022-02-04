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

package xslttransformation

import (
	"context"
	"fmt"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	pkgreconciler "knative.dev/pkg/reconciler"
	"knative.dev/pkg/resolver"

	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/flow/v1alpha1/xslttransformation"
	libreconciler "github.com/triggermesh/triggermesh/pkg/flow/reconciler"
)

// Reconciler implements controller.Reconciler for the component type.
type reconciler struct {
	// adapter properties
	adapterCfg *adapterConfig

	// Knative Service reconciler
	ksvcr libreconciler.KServiceReconciler

	sinkResolver *resolver.URIResolver
}

// Check that our Reconciler implements Interface
var _ reconcilerv1alpha1.Interface = (*reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *reconciler) ReconcileKind(ctx context.Context, o *v1alpha1.XSLTTransformation) pkgreconciler.Event {
	var url *apis.URL
	if o.Spec.Sink != nil {
		var err error
		url, err = r.resolveDestination(ctx, o)
		if err != nil {
			return fmt.Errorf("cannot resolve Sink destination: %w", err)
		}
	}

	ksvc, err := makeAdapterKService(o, r.adapterCfg, url)
	if err != nil {
		o.Status.MarkNotDeployed(v1alpha1.XSLTTransformationReasonWrongSpec, "Cannot create adapter from spec")
		return controller.NewPermanentError(fmt.Errorf("could not make the desired knative service adapter based on the spec: %w", err))
	}

	adapter, event := r.ksvcr.ReconcileKService(ctx, o, ksvc)
	o.Status.PropagateAvailability(adapter)

	return event
}

func (r *reconciler) resolveDestination(ctx context.Context, s *v1alpha1.XSLTTransformation) (*apis.URL, error) {
	dest := s.Spec.Sink.DeepCopy()
	if dest.Ref != nil {
		if dest.Ref.Namespace == "" {
			dest.Ref.Namespace = s.GetNamespace()
		}
	}
	return r.sinkResolver.URIFromDestinationV1(ctx, *dest, s)
}
