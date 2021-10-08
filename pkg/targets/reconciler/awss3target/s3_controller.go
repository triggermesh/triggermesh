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

package awss3target

import (
	"context"

	"github.com/kelseyhightower/envconfig"

	"k8s.io/client-go/tools/cache"
	"knative.dev/eventing/pkg/reconciler/source"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	awss3targetinformer "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/targets/v1alpha1/awss3target"
	"github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/awss3target"
	libreconciler "github.com/triggermesh/triggermesh/pkg/targets/reconciler"

	kserviceclient "knative.dev/serving/pkg/client/injection/client"
	kserviceinformer "knative.dev/serving/pkg/client/injection/informers/serving/v1/service"
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

	serviceInformer := kserviceinformer.Get(ctx)

	impl := awss3target.NewImpl(ctx, &Reconciler{
		ksvcr:      libreconciler.NewKServiceReconciler(kserviceclient.Get(ctx), serviceInformer.Lister()),
		vg:         libreconciler.NewValueGetter(kubeclient.Get(ctx)),
		adapterCfg: adapterCfg,
	})
	logging.FromContext(ctx).Info("Setting up event handlers")

	awsTargetInformer := awss3targetinformer.Get(ctx)
	awsTargetInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	serviceInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterControllerGK(v1alpha1.Kind("AWSS3Target")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	return impl
}
