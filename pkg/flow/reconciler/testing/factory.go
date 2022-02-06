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

package testing

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"knative.dev/pkg/apis/duck"
	"knative.dev/pkg/client/injection/ducks/duck/v1/addressable"
	fakek8sinjectionclient "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/controller"
	fakedynamicclient "knative.dev/pkg/injection/clients/dynamicclient/fake"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/reconciler"
	rt "knative.dev/pkg/reconciler/testing"
	fakeservinginjectionclient "knative.dev/serving/pkg/client/injection/client/fake"

	faketminjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
)

// Ctor constructs a controller.Reconciler.
type Ctor func(*testing.T, context.Context, *Listers) controller.Reconciler

// MakeFactory creates a testing factory for our controller.Reconciler, and
// initializes a Reconciler using the given Ctor as part of the process.
func MakeFactory(ctor Ctor) rt.Factory {
	return func(t *testing.T, tr *rt.TableRow) (controller.Reconciler, rt.ActionRecorderList, rt.EventList) {
		scheme := NewScheme()

		ls := NewListers(scheme, tr.Objects)

		// enable values injected by the test case (TableRow) to be consumed in ctor
		ctx := tr.Ctx
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = logging.WithLogger(ctx, logtesting.TestLogger(t))

		// the controller.Reconciler uses an internal client to handle
		// component objects
		ctx, xsltclient := faketminjectionclient.With(ctx, ls.GetXSLTTransformationObjects()...)

		// all clients used inside reconciler implementations should be
		// injected as well
		ctx, k8sClient := fakek8sinjectionclient.With(ctx, ls.GetKubeObjects()...)
		ctx, servingClient := fakeservinginjectionclient.With(ctx, ls.GetServingObjects()...)

		const eventRecorderBufferSize = 10
		eventRecorder := record.NewFakeRecorder(eventRecorderBufferSize)
		ctx = controller.WithEventRecorder(ctx, eventRecorder)

		// duck informers (e.g. used by resolver.URIResolver) use a dynamic client.
		ctx, _ = fakedynamicclient.With(ctx, scheme, ToUnstructured(t, tr.Objects)...)

		// inject duck informer for Addressables, required by resolver.URIResolver.
		// reconcilertesting.SetupFakeContext would also call this, but
		// we don't need the other informers it registers.
		ctx = addressable.WithDuck(ctx)

		// set up Reconciler from fakes
		r := ctor(t, ctx, &ls)

		// If the reconcilers is leader aware, then promote it.
		if la, ok := r.(reconciler.LeaderAware); ok {
			//nolint:golint,errcheck
			la.Promote(reconciler.UniversalBucket(), func(reconciler.Bucket, types.NamespacedName) {})
		}

		// inject reactors from table row
		for _, reactor := range tr.WithReactors {
			xsltclient.PrependReactor("*", "*", reactor)
			k8sClient.PrependReactor("*", "*", reactor)
			servingClient.PrependReactor("*", "*", reactor)
		}

		actionRecorderList := rt.ActionRecorderList{
			// TriggerMesh clients are merely used here to record status updates,
			// reconcilers don't create or update these objects otherwise.
			xsltclient,

			k8sClient,
			servingClient,
		}

		eventList := rt.EventList{
			Recorder: eventRecorder,
		}

		return r, actionRecorderList, eventList
	}
}

// ToUnstructured takes a list of k8s resources and converts them to
// Unstructured objects.
// We must pass objects as Unstructured to the dynamic client fake, or it
// won't handle them properly.
func ToUnstructured(t *testing.T, objs []runtime.Object) (unstr []runtime.Object) {
	for _, obj := range objs {
		// Only objects that implement duck.OneOfOurs are considered so we can use the duck.ToUnstructured
		// convenience function. This is sufficient for us because the only Unstructured objects we need to
		// handle in tests are Addressables.
		iface, ok := obj.(duck.OneOfOurs)
		if !ok {
			continue
		}

		unstrObj, err := duck.ToUnstructured(iface)
		if err != nil {
			t.Fatalf("Failed to convert object to Unstructured: %s", err)
		}

		unstr = append(unstr, unstrObj)
	}

	return unstr
}
