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

package transformation

import (
	"context"

	"github.com/kelseyhightower/envconfig"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/resolver"
	"knative.dev/pkg/tracker"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1client "knative.dev/serving/pkg/client/injection/client"
	knsvcinformer "knative.dev/serving/pkg/client/injection/informers/serving/v1/service"

	informerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/flow/v1alpha1/transformation"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/flow/v1alpha1/transformation"
)

type envConfig struct {
	Image string `envconfig:"TRANSFORMER_IMAGE" default:"gcr.io/triggermesh/transformation-adapter"`
}

// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	logger := logging.FromContext(ctx)

	transformationInformer := informerv1alpha1.Get(ctx)
	knsvcInformer := knsvcinformer.Get(ctx)

	r := &Reconciler{
		servingClientSet: servingv1client.Get(ctx),
		knServiceLister:  knsvcInformer.Lister(),
	}

	env := &envConfig{}
	envconfig.MustProcess("", env)

	r.transformerImage = env.Image

	impl := reconcilerv1alpha1.NewImpl(ctx, r)
	r.Tracker = tracker.New(impl.EnqueueKey, controller.GetTrackerLease(ctx))

	r.sinkResolver = resolver.NewURIResolverFromTracker(ctx, impl.Tracker)

	logger.Info("Setting up event handlers.")

	transformationInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	knsvcInformer.Informer().AddEventHandler(controller.HandleAll(
		// Call the tracker's OnChanged method, but we've seen the objects
		// coming through this path missing TypeMeta, so ensure it is properly
		// populated.
		controller.EnsureTypeMeta(
			r.Tracker.OnChanged,
			servingv1.SchemeGroupVersion.WithKind("Service"),
		),
	))

	return impl
}
