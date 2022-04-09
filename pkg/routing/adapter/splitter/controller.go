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

package splitter

import (
	"context"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	pkgcontroller "knative.dev/pkg/controller"

	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/routing/v1alpha1/splitter"
	"github.com/triggermesh/triggermesh/pkg/routing/adapter/common/controller"
)

// NewController returns a constructor for the Router's Reconciler.
//
// NOTE(antoineco): although the returned controller doesn't do anything, it is
// necessary to return a valid implementation in order to trigger the Informer
// injection in Knative's sharedmain.Main.
func NewController(component string) pkgadapter.ControllerConstructor {
	return func(ctx context.Context, _ pkgadapter.Adapter) *pkgcontroller.Impl {
		r := (*Reconciler)(nil)
		impl := reconcilerv1alpha1.NewImpl(ctx, r, controller.Opts(component))

		return impl
	}
}
