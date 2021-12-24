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

package ducktypes

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// WaitUntilReady waits until the given resource's status becomes ready.
func WaitUntilReady(c dynamic.Interface, obj *unstructured.Unstructured) *unstructured.Unstructured {
	fieldSelector := fields.OneTermEqualSelector("metadata.name", obj.GetName()).String()
	gvr, _ := meta.UnsafeGuessKindToResource(obj.GroupVersionKind())

	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return c.Resource(gvr).Namespace(obj.GetNamespace()).List(context.Background(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return c.Resource(gvr).Namespace(obj.GetNamespace()).Watch(context.Background(), options)
		},
	}

	// checks whether the object referenced in the given watch.Event has
	// its Ready condition set to True.
	var isResourceReady watchtools.ConditionFunc = func(e watch.Event) (bool, error) {
		if e.Type == watch.Deleted {
			return false, apierrors.NewNotFound(gvr.GroupResource(), obj.GetName())
		}

		if u, ok := e.Object.(*unstructured.Unstructured); ok {
			res := &duckv1.KResource{}
			if err := duck.FromUnstructured(u, res); err != nil {
				framework.FailfWithOffset(2, "Failed to convert unstructured object to KResource: %s", err)
			}

			if cond := res.Status.GetCondition(apis.ConditionReady); cond != nil && cond.IsTrue() {
				return true, nil
			}
		}

		return false, nil
	}

	// NOTE(antoineco): Use a long timeout to make up for long propagation
	// delays of imagePullSecrets to Kubernetes ServiceAccounts by
	// "imagepullsecret-patcher" (triggermesh/triggermesh#101).
	//
	// This value shouldn't exceed CircleCI's "no_output_timeout" to ensure
	// the E2E process doesn't get killed due to inactivity.
	const watchTimeout = 15 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), watchTimeout)
	defer cancel()

	lastEvent, err := watchtools.UntilWithSync(ctx, lw, obj, nil, isResourceReady)
	if err != nil {
		framework.FailfWithOffset(2, "Error waiting for resource %s %q to become ready: %s",
			gvr.GroupResource(), obj.GetName(), err)
	}

	return lastEvent.Object.(*unstructured.Unstructured)
}
