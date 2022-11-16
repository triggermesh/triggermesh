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

package kafkasource

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	rt "knative.dev/pkg/reconciler/testing"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/kafkasource"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	. "github.com/triggermesh/triggermesh/pkg/reconciler/testing"
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
			adapterCfg: cfg,
		}

		r.base = NewTestDeploymentReconciler[*v1alpha1.KafkaSource](ctx, ls,
			ls.GetKafkaSourceLister().KafkaSources,
		)

		return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
			fakeinjectionclient.Get(ctx), ls.GetKafkaSourceLister(),
			controller.GetEventRecorder(ctx), r)
	}
}

// newEventSource returns a test source object with a minimal set of pre-filled attributes.
func newEventSource() *v1alpha1.KafkaSource {
	securityMechanism := "PLAIN"
	username := "admin"

	src := &v1alpha1.KafkaSource{
		Spec: v1alpha1.KafkaSourceSpec{
			BootstrapServers: []string{"localhost:9092"},
			Topic:            "test-topic",
			GroupID:          "test-consumer-group",
			Auth: v1alpha1.KafkaSourceAuth{
				SASLEnable:         true,
				SecurityMechanisms: &securityMechanism,
				Username:           &username,
				Password: &commonv1alpha1.ValueFromField{
					ValueFromSecret: &corev1.SecretKeySelector{
						Key: "key1",
					},
				},
			},
		},
	}

	Populate(src)

	return src
}

// adapterBuilder returns a slim Reconciler containing only the fields accessed
// by r.BuildAdapter().
func adapterBuilder(cfg *adapterConfig) common.AdapterBuilder[*appsv1.Deployment] {
	return &Reconciler{
		adapterCfg: cfg,
	}
}
