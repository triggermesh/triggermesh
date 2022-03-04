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
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// GetPodLogs returns a stream of the logs just like GetLogs, but requires an explicit pod name.
func GetPodLogs(c clientset.Interface, namespace, podName string) io.ReadCloser {
	pod, err := c.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		framework.FailfWithOffset(2, "Failed to get Pod: %s", err)
	}

	logStream, err := c.CoreV1().Pods(namespace).
		GetLogs(pod.Name, &corev1.PodLogOptions{Container: pod.Spec.Containers[0].Name}).Stream(context.Background())
	if err != nil {
		framework.FailfWithOffset(2, "Failed to stream logs of Pod %s: %s", pod.Name, err)
	}

	return logStream
}
