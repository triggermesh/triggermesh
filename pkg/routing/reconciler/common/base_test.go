/*
Copyright (c) 2021 TriggerMesh Inc.

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

package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/kmeta"
	logtesting "knative.dev/pkg/logging/testing"
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

	impl := controller.NewImplFull(nil, controller.ControllerOptions{
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
