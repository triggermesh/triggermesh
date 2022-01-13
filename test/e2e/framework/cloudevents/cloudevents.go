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

// Package cloudevents contains helpers to interact with CloudEvents.
package cloudevents

import (
	"context"
	"encoding/json"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol/http"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// E2ECeExtension is a CloudEvent extension that is used to propagate
// the name of the test that generated the event.
const E2ECeExtension = "iotriggermeshe2e"

const (
	curlContainerName     = "curl"
	curlContainerImage    = "curlimages/curl:latest"
	eventSenderNamePrefix = "eventsender-"
)

// NewHelloEvent generates a CloudEvent with dummy values.
func NewHelloEvent(f *framework.Framework) *cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetID("0000")
	event.SetType("e2e.test")
	event.SetSource("e2e.triggermesh")
	event.SetExtension(E2ECeExtension, f.UniqueName)
	if err := event.SetData(cloudevents.TextPlain, "Hello, World"); err != nil {
		framework.FailfWithOffset(2, "Error setting event data: %s", err)
	}

	return &event
}

// RunEventSender runs a job which sends a CloudEvent payload to the given URL.
// The function doesn't wait for the job to complete, but the job gets retried
// in case of failure.
func RunEventSender(c clientset.Interface, namespace, url string, payload *cloudevents.Event) *batchv1.Job {
	eventJSON, err := json.Marshal(payload)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to serialize CloudEvent to JSON: %s", err)
	}

	job := makeCurlJob(namespace, []string{
		"-s",  // hide progress meter
		"-S",  // show errors
		"-D-", // dump headers to stdout

		// In Structured Content Mode, the entire payload is sent in the request body.
		// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#32-structured-content-mode
		"-H", http.ContentType + ": " + cloudevents.ApplicationCloudEventsJSON,
		"--data-raw", string(eventJSON),

		url,
	})

	job, err = c.BatchV1().Jobs(namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create Job: %s", err)
	}

	return job
}

// makeCurlJob returns a Job object that runs a cURL command.
func makeCurlJob(namespace string, args []string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: eventSenderNamePrefix,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  curlContainerName,
						Image: curlContainerImage,
						Args:  args,
					}},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}
