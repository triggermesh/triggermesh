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

package function

import (
	"context"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/client/injection/kube/client"
	cminformer "knative.dev/pkg/client/injection/kube/informers/core/v1/configmap"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/resolver"
	"knative.dev/pkg/tracker"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/injection/client"
	knsvcinformer "knative.dev/serving/pkg/client/injection/informers/serving/v1/service"

	informerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/extensions/v1alpha1/function"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/extensions/v1alpha1/function"
)

const (
	eventStoreEnv    = "EVENTSTORE_URI"
	runtimeEnvPrefix = "RUNTIME_"
)

// New creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	logger := logging.FromContext(ctx)

	functionInformer := informerv1alpha1.Get(ctx)
	knSvcInformer := knsvcinformer.Get(ctx)
	cmInformer := cminformer.Get(ctx)

	r := &Reconciler{
		coreClientSet:      client.Get(ctx),
		cmLister:           cmInformer.Lister(),
		knServingClientSet: servingv1client.Get(ctx),
		knServiceLister:    knSvcInformer.Lister(),
	}

	impl := reconcilerv1alpha1.NewImpl(ctx, r)

	r.Tracker = tracker.New(impl.EnqueueKey, controller.GetTrackerLease(ctx))
	r.sinkResolver = resolver.NewURIResolver(ctx, impl.EnqueueKey)
	r.runtimes = make(map[string]string)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, runtimeEnvPrefix) {
			continue
		}
		e = strings.TrimPrefix(e, runtimeEnvPrefix)
		runtimePairs := strings.SplitN(e, "=", 2)
		r.runtimes[runtimePairs[0]] = runtimePairs[1]
	}

	logger.Info("Setting up event handlers.")

	functionInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	knSvcInformer.Informer().AddEventHandler(controller.HandleAll(
		controller.EnsureTypeMeta(
			r.Tracker.OnChanged,
			servingv1.SchemeGroupVersion.WithKind("Service"),
		),
	))

	cmInformer.Informer().AddEventHandler(controller.HandleAll(
		controller.EnsureTypeMeta(
			r.Tracker.OnChanged,
			corev1.SchemeGroupVersion.WithKind("ConfigMap"),
		),
	))

	return impl
}
