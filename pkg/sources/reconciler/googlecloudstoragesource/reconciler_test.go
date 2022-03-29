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

package googlecloudstoragesource

import (
	"context"
	"testing"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/ptr"
	rt "knative.dev/pkg/reconciler/testing"

	gpubsub "cloud.google.com/go/pubsub"
	gstorage "cloud.google.com/go/storage"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/googlecloudstoragesource"
	"github.com/triggermesh/triggermesh/pkg/sources/client/gcloud/storage"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	. "github.com/triggermesh/triggermesh/pkg/sources/reconciler/testing"
)

func TestReconcileSource(t *testing.T) {
	adapterCfg := &adapterConfig{
		Image:   "registry/image:tag",
		configs: &source.EmptyVarsGenerator{},
	}

	ctor := reconcilerCtor(adapterCfg)
	src := newEventSource()
	ab := adapterBuilder(adapterCfg)

	TestReconcileAdapter(t, ctor, src, ab)
}

// reconcilerCtor returns a Ctor for a source Reconciler.
func reconcilerCtor(cfg *adapterConfig) Ctor {
	return func(t *testing.T, ctx context.Context, _ *rt.TableRow, ls *Listers) controller.Reconciler {
		r := &Reconciler{
			cg:         staticClientGetter((*gpubsub.Client)(nil), (*gstorage.Client)(nil)),
			srcLister:  ls.GetGoogleCloudStorageSourceLister().GoogleCloudStorageSources,
			base:       NewTestDeploymentReconciler(ctx, ls),
			adapterCfg: cfg,
		}

		return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
			fakeinjectionclient.Get(ctx), ls.GetGoogleCloudStorageSourceLister(),
			controller.GetEventRecorder(ctx), r)
	}
}

// newEventSource returns a test source object with a minimal set of pre-filled attributes.
func newEventSource() *v1alpha1.GoogleCloudStorageSource {
	src := &v1alpha1.GoogleCloudStorageSource{
		Spec: v1alpha1.GoogleCloudStorageSourceSpec{
			Bucket: "my-bucket",
			PubSub: v1alpha1.GoogleCloudStorageSourcePubSubSpec{
				Project: ptr.String("my-project"),
			},
			ServiceAccountKey: v1alpha1.ValueFromField{
				Value: "{}",
			},
		},
	}

	// assume finalizer is already set to prevent the generated reconciler
	// from generating an extra Patch action
	src.Finalizers = []string{sources.GoogleCloudStorageSourceResource.String()}

	Populate(src)

	return src
}

// adapterBuilder returns a slim Reconciler containing only the fields accessed
// by r.BuildAdapter().
func adapterBuilder(cfg *adapterConfig) common.AdapterDeploymentBuilder {
	return &Reconciler{
		adapterCfg: cfg,
	}
}

/* Google Cloud clients */

// staticClientGetter transforms the given client into a ClientGetter.
func staticClientGetter(psCli *gpubsub.Client, stCli *gstorage.Client) storage.ClientGetterFunc {
	return func(*v1alpha1.GoogleCloudStorageSource) (*gpubsub.Client, *gstorage.Client, error) {
		return psCli, stCli, nil
	}
}
