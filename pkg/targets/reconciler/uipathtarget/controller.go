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

package uipathtarget

import (
	"context"

	"github.com/kelseyhightower/envconfig"
	"k8s.io/client-go/tools/cache"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	kserviceclient "knative.dev/serving/pkg/client/injection/client"
	kserviceinformer "knative.dev/serving/pkg/client/injection/informers/serving/v1/service"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	uipathtargetinformer "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/targets/v1alpha1/uipathtarget"
	"github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/uipathtarget"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"
)

// NewController initializes the controller and is called by the generated code
// Registers event handlers to enqueue events
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {

	adapterCfg := &adapterConfig{
		obsConfig: source.WatchConfigurations(ctx, adapterName, cmw, source.WithLogging, source.WithMetrics),
	}

	envconfig.MustProcess("", adapterCfg)

	ksvcInformer := kserviceinformer.Get(ctx)

	r := &reconciler{
		logger:     logging.FromContext(ctx),
		ksvcr:      libreconciler.NewKServiceReconciler(kserviceclient.Get(ctx), ksvcInformer.Lister()),
		adapterCfg: adapterCfg,
	}

	impl := uipathtarget.NewImpl(ctx, r)

	uipathTargetInformer := uipathtargetinformer.Get(ctx)
	uipathTargetInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	ksvcInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterControllerGK(v1alpha1.Kind("UiPathTarget")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	return impl
}
