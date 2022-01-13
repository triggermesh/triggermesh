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

package apps

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// WaitForCompletion waits until the given Job completes.
func WaitForCompletion(c clientset.Interface, j *batchv1.Job) {
	fieldSelector := fields.OneTermEqualSelector("metadata.name", j.Name).String()

	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return c.BatchV1().Jobs(j.Namespace).List(context.Background(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return c.BatchV1().Jobs(j.Namespace).Watch(context.Background(), options)
		},
	}

	gr := schema.GroupResource{Group: "batch", Resource: "jobs"}

	// checks whether the Job referenced in the given watch.Event has completed.
	var isJobComplete watchtools.ConditionFunc = func(e watch.Event) (bool, error) {
		if e.Type == watch.Deleted {
			return false, apierrors.NewNotFound(gr, j.Name)
		}

		if j, ok := e.Object.(*batchv1.Job); ok {
			for _, c := range j.Status.Conditions {
				switch c.Type {
				case batchv1.JobComplete:
					return c.Status == corev1.ConditionTrue, nil
				case batchv1.JobFailed:
					return false, fmt.Errorf("job %q failed with reason %s: %s",
						j.Name, c.Reason, c.Message)
				}
			}
		}

		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	_, err := watchtools.UntilWithSync(ctx, lw, &batchv1.Job{}, nil, isJobComplete)
	if err != nil {
		framework.FailfWithOffset(2, "Error waiting for %s %q to complete: %s", gr, j.Name, err)
	}
}
