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

package tekton

import (
	"context"
	"fmt"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	watchtools "k8s.io/client-go/tools/watch"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

func WaitForTaskRunCompletion(c tektonclient.Interface, namespace, name string) {
	fieldSelector := fields.OneTermEqualSelector("metadata.name", name).String()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return c.TektonV1beta1().TaskRuns(namespace).List(context.Background(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return c.TektonV1beta1().TaskRuns(namespace).Watch(context.Background(), opts)
		},
	}

	var isFinished watchtools.ConditionFunc = func(e watch.Event) (bool, error) {
		if r, ok := e.Object.(*v1beta1.TaskRun); ok {
			for _, c := range r.Status.Conditions {
				// NOTE: The type will always be 'Succeeded' so we need to check status or reason
				// Also, skip unknown status as that indicates the run is in progress
				if c.Status == corev1.ConditionTrue {
					return true, nil
				} else if c.Status == corev1.ConditionFalse {
					return false, fmt.Errorf("taskrun %s failed with reason %s: %s", name, c.Reason, c.Message)
				}
			}
		}

		return false, nil
	}

	// Default the context timeout to 1 minute
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	_, err := watchtools.UntilWithSync(ctx, lw, &v1beta1.TaskRun{}, nil, isFinished)
	if err != nil {
		framework.FailfWithOffset(2, "Error waiting for %s to complete: %s", name, err)
	}
}

func WaitForPipelineRunCompletion(c tektonclient.Interface, namespace, name string) {
	fieldSelector := fields.OneTermEqualSelector("metadata.name", name).String()

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return c.TektonV1beta1().PipelineRuns(namespace).List(context.Background(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return c.TektonV1beta1().PipelineRuns(namespace).Watch(context.Background(), opts)
		},
	}

	var isFinished watchtools.ConditionFunc = func(e watch.Event) (bool, error) {
		if r, ok := e.Object.(*v1beta1.PipelineRun); ok {
			for _, c := range r.Status.Conditions {
				// NOTE: The type will always be 'Succeeded' so we need to check status or reason
				// Also, skip unknown status as that indicates the run is in progress
				if c.Status == corev1.ConditionTrue {
					return true, nil
				} else if c.Status == corev1.ConditionFalse {
					return false, fmt.Errorf("pipelinerun %s failed with reason %s: %s", name, c.Reason, c.Message)
				}
			}
		}

		return false, nil
	}

	// Default the context timeout to 1 minute
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	_, err := watchtools.UntilWithSync(ctx, lw, &v1beta1.PipelineRun{}, nil, isFinished)
	if err != nil {
		framework.FailfWithOffset(2, "Error waiting for %s to complete: %s", name, err)
	}
}
