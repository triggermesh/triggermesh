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

package tektontarget

import (
	"context"

	"github.com/kelseyhightower/envconfig"
	"k8s.io/client-go/tools/cache"

	"knative.dev/eventing/pkg/reconciler/source"
	k8sclient "knative.dev/pkg/client/injection/kube/client"
	serviceaccountinformerv1 "knative.dev/pkg/client/injection/kube/informers/core/v1/serviceaccount"
	rolebindinginformerv1 "knative.dev/pkg/client/injection/kube/informers/rbac/v1/rolebinding"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	kserviceclient "knative.dev/serving/pkg/client/injection/client"
	kserviceinformer "knative.dev/serving/pkg/client/injection/informers/serving/v1/service"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	informerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/targets/v1alpha1/tektontarget"
	"github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/targets/v1alpha1/tektontarget"
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

	targetInformer := informerv1alpha1.Get(ctx)
	serviceInformer := kserviceinformer.Get(ctx)

	r := &reconciler{
		logger:       logging.FromContext(ctx),
		ksvcr:        libreconciler.NewKServiceReconciler(kserviceclient.Get(ctx), serviceInformer.Lister()),
		vg:           libreconciler.NewValueGetter(k8sclient.Get(ctx)),
		adapterCfg:   adapterCfg,
		saClient:     k8sclient.Get(ctx).CoreV1().ServiceAccounts,
		rbClient:     k8sclient.Get(ctx).RbacV1().RoleBindings,
		targetLister: targetInformer.Lister().TektonTargets,
		saLister:     serviceaccountinformerv1.Get(ctx).Lister().ServiceAccounts,
		rbLister:     rolebindinginformerv1.Get(ctx).Lister().RoleBindings,
	}

	impl := tektontarget.NewImpl(ctx, r)
	logging.FromContext(ctx).Info("Setting up event handlers")

	targetInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	serviceInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterControllerGK(v1alpha1.Kind("TektonTarget")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	// Spawn a thread to reap stale Tekton run objects
	go reaperThread(ctx, r)

	return impl
}
