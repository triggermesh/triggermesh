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

package azureiothubsource

import (
	"context"

	"github.com/kelseyhightower/envconfig"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	informerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/sources/v1alpha1/azureiothubsource"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/azureiothubsource"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
)

// NewController initializes the controller and is called by the generated code
// Registers event handlers to enqueue events
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	typ := (*v1alpha1.AzureIOTHubSource)(nil)
	app := common.ComponentName(typ)
	adapterCfg := &adapterConfig{
		configs: source.WatchConfigurations(ctx, app, cmw, source.WithLogging, source.WithMetrics),
	}

	envconfig.MustProcess(app, adapterCfg)
	informer := informerv1alpha1.Get(ctx)
	r := &Reconciler{
		adapterCfg: adapterCfg,
		srcLister:  informer.Lister().AzureIOTHubSources,
	}

	impl := reconcilerv1alpha1.NewImpl(ctx, r)
	r.base = common.NewGenericDeploymentReconciler(
		ctx,
		typ.GetGroupVersionKind(),
		impl.Tracker,
		impl.EnqueueControllerOf,
	)

	informer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
	return impl
}
