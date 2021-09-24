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

package splitter

import (
	"context"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	pkgcontroller "knative.dev/pkg/controller"

	"github.com/triggermesh/triggermesh/pkg/apis/routing/v1alpha1"
	informerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/routing/v1alpha1/splitter"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/routing/v1alpha1/splitter"
	"github.com/triggermesh/triggermesh/pkg/routing/adapter/common/controller"
)

// MTAdapter allows the multi-tenant adapter to expose methods the reconciler
// can call while reconciling a source object.
type MTAdapter interface {
	// Registers a HTTP handler for the given source.
	RegisterHandlerFor(context.Context, *v1alpha1.Splitter) error
	// Deregisters the HTTP handler for the given source.
	DeregisterHandlerFor(context.Context, *v1alpha1.Splitter) error
}

// NewController returns a constructor for the event source's Reconciler.
func NewController(component string) pkgadapter.ControllerConstructor {
	return func(ctx context.Context, a pkgadapter.Adapter) *pkgcontroller.Impl {
		r := &Reconciler{
			adapter: a.(MTAdapter),
		}
		impl := reconcilerv1alpha1.NewImpl(ctx, r, controller.Opts(component))

		informerv1alpha1.Get(ctx).Informer().AddEventHandler(pkgcontroller.HandleAll(impl.Enqueue))

		return impl
	}
}
