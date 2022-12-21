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

package reconciler

import (
	"context"
	"testing"
	"time"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	fakek8sclient "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/kmeta"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/ptr"
	rectesting "knative.dev/pkg/reconciler/testing"

	// Link fake informers accessed by the code under test
	_ "knative.dev/pkg/client/injection/kube/informers/apps/v1/deployment/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/apps/v1/replicaset/fake"
)

func TestEnqueueObjectsInNamespaceOf(t *testing.T) {
	const ns1, ns2 = "ns1", "ns2"

	storedObjects := []interface{}{
		newFakeObject(ns1, "foo"),
		newFakeObject(ns1, "bar"),
		newFakeObject(ns1, "baz"),
		newFakeObject(ns2, "qux"),
	}

	inf := &fakeInformer{
		store: storedObjects,
	}

	impl := controller.NewContext(context.Background(), nil, controller.ControllerOptions{
		Logger: logtesting.TestLogger(t),
	})
	t.Cleanup(impl.WorkQueue().ShutDown)

	handlerFn := EnqueueObjectsInNamespaceOf(inf, impl.FilteredGlobalResync, nil)

	// handling an object in ns1 should enqueue foo,bar,baz but not qux
	handlerFn(
		newFakeObject(ns1, "fake-adapter"),
	)

	// account for filtering/processing latency
	time.Sleep(10 * time.Millisecond)

	expectEnqueued := []string{"ns1/foo", "ns1/bar", "ns1/baz"}

	require.Equal(t, len(expectEnqueued), impl.WorkQueue().Len(), "Unexpected queue length")

	enqueued := popKeys(len(expectEnqueued), impl.WorkQueue())
	assert.ElementsMatch(t, expectEnqueued, enqueued)
}

func TestAdapterOverrideOptions(t *testing.T) {
	t.Run("overrideOptions", func(t *testing.T) {
		objectOptions := adapterOverrideOptions(&v1alpha1.AdapterOverrides{Env: []corev1.EnvVar{{Name: "foo", Value: "bar"}}})
		assert.Len(t, objectOptions, 2) // bad test
	})
}

func TestHasAdapterLabelsForType(t *testing.T) {
	typ := &fakeObject{}
	filterFn := hasAdapterLabelsForType(&fakeObject{})

	t.Run("matches common selector", func(t *testing.T) {
		objAllLabels := &fakeObject{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					appNameLabel:      ComponentName(typ),
					appComponentLabel: componentAdapter,
					appPartOfLabel:    partOf,
					appManagedByLabel: managedBy,
					"extra":           "label",
				},
			},
		}

		assert.True(t, filterFn(objAllLabels), "Expected to match common set of labels")
	})

	t.Run("doesn't match common selector", func(t *testing.T) {
		objMissingLabels := &fakeObject{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					appPartOfLabel:    partOf,
					appManagedByLabel: managedBy,
				},
			},
		}

		assert.False(t, filterFn(objMissingLabels), "Expected not to match common set of labels")
	})
}

func TestOuterMostAncestorControllerRef(t *testing.T) {
	const tNs = "test"

	testCases := map[string]struct {
		objects []runtime.Object
		expect  *metav1.OwnerReference
	}{
		/* In this test, the outermost ancestor of kind Foo is resolved
		   all the way through a ReplicaSet and a Deployment.
		*/
		"Recurse lookup through multiple ancestors": {
			objects: []runtime.Object{
				&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
					Namespace: tNs,
					Name:      "foo-fake-adapter",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: "fake/v0",
						Kind:       "Foo",
						Name:       "fake",
						Controller: ptr.Bool(true),
					}},
				}},
				&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
					Namespace: tNs,
					Name:      "foo-fake-adapter-abc012",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: appsv1.SchemeGroupVersion.String(),
						Kind:       "Deployment",
						Name:       "foo-fake-adapter",
						Controller: ptr.Bool(true),
					}},
				}},
				&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Namespace: tNs,
					Name:      "foo-fake-adapter-abc012-efg345",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: appsv1.SchemeGroupVersion.String(),
						Kind:       "ReplicaSet",
						Name:       "foo-fake-adapter-abc012",
						Controller: ptr.Bool(true),
					}},
				}},
			},
			expect: &metav1.OwnerReference{
				APIVersion: "fake/v0",
				Kind:       "Foo",
				Name:       "fake",
			},
		},
		/* In this test, an ancestor of kind ReplicaSet is found to
		   have no controller, which makes it the outermost ancestor.
		*/
		"Ancestor with no controller": {
			objects: []runtime.Object{
				&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
					Namespace:       tNs,
					Name:            "foo-fake-adapter-abc012",
					OwnerReferences: []metav1.OwnerReference{},
				}},
				&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Namespace: tNs,
					Name:      "foo-fake-adapter-abc012-efg345",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: appsv1.SchemeGroupVersion.String(),
						Kind:       "ReplicaSet",
						Name:       "foo-fake-adapter-abc012",
						Controller: ptr.Bool(true),
					}},
				}},
			},
			expect: &metav1.OwnerReference{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       "foo-fake-adapter-abc012",
			},
		},
		/* In this test, the ReplicaSet is owned by a Deployment, but
		   this Deployment can't be found in the lister so the ancestor
		   is considered unknown.
		*/
		"One supported kind of ancestor doesn't exist": {
			objects: []runtime.Object{
				&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
					Namespace: tNs,
					Name:      "foo-fake-adapter-abc012",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: appsv1.SchemeGroupVersion.String(),
						Kind:       "Deployment",
						Name:       "foo-fake-adapter",
						Controller: ptr.Bool(true),
					}},
				}},
				&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Namespace: tNs,
					Name:      "foo-fake-adapter-abc012-efg345",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: appsv1.SchemeGroupVersion.String(),
						Kind:       "ReplicaSet",
						Name:       "foo-fake-adapter-abc012",
						Controller: ptr.Bool(true),
					}},
				}},
			},
			expect: nil,
		},
		/* In this test, the outermost ancestor of kind Foo can't be
		   reached because one intermediate ancestor is of kind Service,
		   which the recursive resolver doesn't support.
		   The Service is returned as the last resolvable ancestor.
		*/
		"One kind of ancestor is not supported for recursive lookup": {
			objects: []runtime.Object{
				&corev1.Service{ObjectMeta: metav1.ObjectMeta{
					Namespace: tNs,
					Name:      "fake",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: "fake/v0",
						Kind:       "Foo",
						Name:       "fake",
						Controller: ptr.Bool(true),
					}},
				}},
				&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Namespace: tNs,
					Name:      "fake",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: corev1.SchemeGroupVersion.String(),
						Kind:       "Service",
						Name:       "fake",
						Controller: ptr.Bool(true),
					}},
				}},
			},
			expect: &metav1.OwnerReference{
				APIVersion: "v1",
				Kind:       "Service",
				Name:       "fake",
			},
		},
		/* In this test, the object to resolve the ancestor for doesn't
		   have a controller. Because it is the initial object, there
		   is no ancestor to return at all.
		*/
		"No controller": {
			objects: []runtime.Object{
				&corev1.Pod{ObjectMeta: metav1.ObjectMeta{
					Namespace:       tNs,
					Name:            "fake",
					OwnerReferences: []metav1.OwnerReference{},
				}},
			},
			expect: nil,
		},
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			withK8sCli := func(ctx context.Context, _ *rest.Config) context.Context {
				ctx, _ = fakek8sclient.With(ctx, tc.objects...)
				return ctx
			}
			injection.Fake.RegisterClient(withK8sCli)

			stopCh := make(chan struct{})
			t.Cleanup(func() { close(stopCh) })

			ctx, infs := rectesting.SetupFakeContext(t)
			err := controller.StartInformers(stopCh, infs...)
			require.NoError(t, err)

			objectToResolve := tc.objects[len(tc.objects)-1].(metav1.Object)

			ancestorCtlrRef := outermostAncestorControllerRef(ctx, objectToResolve)

			if tc.expect == nil {
				assert.Nil(t, ancestorCtlrRef)
			} else {
				assert.Equal(t, tc.expect.APIVersion, ancestorCtlrRef.APIVersion)
				assert.Equal(t, tc.expect.Kind, ancestorCtlrRef.Kind)
				assert.Equal(t, tc.expect.Name, ancestorCtlrRef.Name)
			}
		})
	}
}

// popKeys pops n items from a queue and returns their keys.
func popKeys(n int, q workqueue.RateLimitingInterface) []string {
	enqueuedObjs := make([]string, n)

	for i := range enqueuedObjs {
		obj, _ := q.Get()
		enqueuedObjs[i] = obj.(types.NamespacedName).String()
	}

	return enqueuedObjs
}

// fakeObject implements kmeta.Accessor and kmeta.OwnerRefable for tests.
type fakeObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	runtime.Object
}

var (
	_ kmeta.Accessor     = (*fakeObject)(nil)
	_ kmeta.OwnerRefable = (*fakeObject)(nil)
)

// disambiguate GetObjectKind method contained in by both runtime.Object and metav1.TypeMeta.
func (*fakeObject) GetObjectKind() schema.ObjectKind { return nil }

// GetGroupVersionKind implements kmeta.OwnerRefable.
func (*fakeObject) GetGroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Kind: "Fake",
	}
}

// newFakeObject creates a fake API object with the given namespace and name.
func newFakeObject(ns, name string) metav1.Object {
	return &fakeObject{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}
}

// fakeInformer mocks a cache.SharedInformer.
type fakeInformer struct {
	store fakeStore
}

// GetStore implements cache.SharedInformer.
func (i *fakeInformer) GetStore() cache.Store {
	return i.store
}

// fakeStore mocks a cache.Store.
type fakeStore []interface{}

// List implements cache.Store.
func (s fakeStore) List() []interface{} {
	return s
}

/* unimplemented in those tests */

func (*fakeInformer) AddEventHandler(cache.ResourceEventHandler) {}

func (*fakeInformer) AddEventHandlerWithResyncPeriod(cache.ResourceEventHandler, time.Duration) {}

func (*fakeInformer) GetController() cache.Controller { return nil }

func (*fakeInformer) Run(<-chan struct{}) {}

func (*fakeInformer) HasSynced() bool { return true }

func (*fakeInformer) LastSyncResourceVersion() string { return "" }

func (*fakeInformer) SetWatchErrorHandler(cache.WatchErrorHandler) error { return nil }

func (fakeStore) Add(interface{}) error { return nil }

func (fakeStore) Update(interface{}) error { return nil }

func (fakeStore) Delete(interface{}) error { return nil }

func (fakeStore) ListKeys() []string { return nil }

func (fakeStore) Get(interface{}) (interface{}, bool, error) { return nil, false, nil }

func (fakeStore) GetByKey(string) (interface{}, bool, error) { return nil, false, nil }

func (fakeStore) Replace([]interface{}, string) error { return nil }

func (fakeStore) Resync() error { return nil }
