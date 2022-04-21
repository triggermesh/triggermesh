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

package function

import (
	"context"
	"testing"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	rt "knative.dev/pkg/reconciler/testing"

	"github.com/triggermesh/triggermesh/pkg/apis/extensions/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/extensions/v1alpha1/function"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	. "github.com/triggermesh/triggermesh/pkg/reconciler/testing"
)

func TestReconcile(t *testing.T) {
	t.Setenv("RUNTIME_PYTHON", "registry/image:tag")

	adapterCfg := &adapterConfig{
		obsConfig: &source.EmptyVarsGenerator{},
	}

	ctor := reconcilerCtor(adapterCfg)
	trg := newFunction()
	ab := adapterBuilder(adapterCfg)

	TestReconcileAdapter(t, ctor, trg, ab)
}

// reconcilerCtor returns a Ctor for a Function Reconciler.
func reconcilerCtor(cfg *adapterConfig) Ctor {
	return func(t *testing.T, ctx context.Context, _ *rt.TableRow, ls *Listers) controller.Reconciler {
		r := &Reconciler{
			adapterCfg: cfg,
		}

		r.base = NewTestServiceReconciler(
			ctx,
			ls,
			func(namespace string) common.Lister[*v1alpha1.Function] {
				return ls.GetFunctionLister().Functions(namespace)
			},
		)

		return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
			fakeinjectionclient.Get(ctx), ls.GetFunctionLister(),
			controller.GetEventRecorder(ctx), r)
	}
}

// newFunction returns a populated Function object.
func newFunction() *v1alpha1.Function {
	fn := &v1alpha1.Function{
		Spec: v1alpha1.FunctionSpec{
			Runtime: "python",
		},
	}

	Populate(fn)

	return fn
}

// adapterBuilder returns a slim Reconciler containing only the fields accessed
// by r.BuildAdapter().
func adapterBuilder(cfg *adapterConfig) common.AdapterServiceBuilder {
	return &Reconciler{
		adapterCfg: cfg,
	}
}
