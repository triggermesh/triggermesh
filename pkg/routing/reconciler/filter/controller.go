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

package filter

import (
	"context"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"

	"github.com/kelseyhightower/envconfig"
	"github.com/triggermesh/triggermesh/pkg/apis/routing/v1alpha1"
	informerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/routing/v1alpha1/filter"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/routing/v1alpha1/filter"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
)

// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {

	typ := (*v1alpha1.Filter)(nil)
	app := common.ComponentName(typ)

	// Calling envconfig.Process() with a prefix appends that prefix
	// (uppercased) to the Go field name, e.g. MYSOURCE_IMAGE.
	adapterCfg := &adapterConfig{
		configs: source.WatchConfigurations(ctx, app, cmw, source.WithLogging, source.WithMetrics),
	}
	envconfig.MustProcess(app, adapterCfg)

	informer := informerv1alpha1.Get(ctx)

	r := &Reconciler{
		adapterCfg: adapterCfg,
	}
	impl := reconcilerv1alpha1.NewImpl(ctx, r)

	logger := logging.FromContext(ctx)

	r.base = common.NewMTGenericServiceReconciler[*v1alpha1.Filter](
		ctx,
		typ,
		impl.Tracker,
		common.EnqueueObjectsInNamespaceOf(informer.Informer(), impl.FilteredGlobalResync, logger),
		informer.Lister().Filters,
	)

	informer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	return impl
}
