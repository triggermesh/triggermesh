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

package googlecloudstorage

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo" //nolint:stylecheck
	. "github.com/onsi/gomega" //nolint:stylecheck

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"

	duckv1 "knative.dev/pkg/apis/duck/v1"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	"github.com/triggermesh/triggermesh/test/e2e/framework/bridges"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
	e2egcloud "github.com/triggermesh/triggermesh/test/e2e/framework/gcloud"
	e2estorage "github.com/triggermesh/triggermesh/test/e2e/framework/gcloud/storage"
)

/* This test suite requires:

   - A Google Cloud Service Account key in JSON format, exported in the environment as GCLOUD_SERVICEACCOUNT_KEY
   - The name of the Google Cloud project exported in the environment as GCLOUD_PROJECT
*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "GoogleCloudStorageSource"
	sourceResource = "googlecloudstoragesources"

	credsEnvVar   = "GCLOUD_SERVICEACCOUNT_KEY"
	projectEnvVar = "GCLOUD_PROJECT"
)

var _ = Describe("Google Cloud Storage source", func() {
	// Storage buckets aren't allowed to contain "google"
	f := framework.New("cloudstoragesource")

	var ns string

	var srcClient dynamic.ResourceInterface

	var bucket string
	var object string
	var project string
	var saKey string

	var sink *duckv1.Destination

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := sourceAPIVersion.WithResource(sourceResource)
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source watches a bucket", func() {
		var storageClient *storage.Client
		var src *unstructured.Unstructured
		var err error

		BeforeEach(func() {
			saKey = e2egcloud.GetCreds(credsEnvVar)
			project = e2egcloud.GetProject(projectEnvVar)
			storageClient, err = storage.NewClient(context.Background(), option.WithCredentialsJSON([]byte(saKey)))
			Expect(err).ToNot(HaveOccurred())

			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
			})

			By("creating a bucket", func() {
				bucket = e2estorage.CreateBucket(storageClient, project, f)
			})

			By("creating a GoogleCloudStorage object", func() {
				src, err = createSource(srcClient, ns, "test-", sink,
					withBucket(bucket),
					withProject(project),
					withEventTypes("OBJECT_FINALIZE"),
					withCredentials(saKey),
				)
				Expect(err).ToNot(HaveOccurred())
				ducktypes.WaitUntilReady(f.DynamicClient, src)
			})
		})

		AfterEach(func() {
			By("deleting storage object "+object, func() {
				e2estorage.DeleteObject(storageClient, bucket, object)
			})
			By("deleting storage bucket "+bucket, func() {
				e2estorage.DeleteBucket(storageClient, bucket)
			})
			By("deleting a GoogleCloudStorage object", func() {
				err := srcClient.Delete(context.Background(), src.GetName(), metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("a new object is created", func() {
			BeforeEach(func() {
				object = e2estorage.CreateObject(storageClient, project, bucket, f)
			})

			Specify("the source generates an event", func() {
				const receiveTimeout = 15 * time.Second
				const pollInterval = 500 * time.Millisecond

				var receivedEvents []cloudevents.Event

				readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

				Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
				Expect(receivedEvents).To(HaveLen(1))

				e := receivedEvents[0]

				Expect(e.Type()).To(Equal("com.google.cloud.storage.notification"))
				Expect(e.Source()).To(Equal("gs://" + bucket))
			})
		})
	})

	When("a client creates a source object with invalid event Types", func() {

		// Those tests do not require a real repository or sink
		BeforeEach(func() {
			saKey = "fake-creds"

			sink = &duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: "fake/v1",
					Kind:       "Fake",
					Name:       "fake",
				},
			}
		})

		// Here we use
		//   "Specify: the API server rejects ..., By: setting an invalid ..."
		// instead of
		//   "When: it sets an invalid ..., Specify: the API server rejects ..."
		// to avoid creating a namespace for each spec, due to their simplicity.
		Specify("the API server rejects the creation of that object", func() {

			By("setting an invalid bucket", func() {
				bucket := "fake-bucket"

				_, err := createSource(srcClient, ns, "test-invalid-eventTypes-", sink,
					withBucket(bucket),
					withProject(project),
					withEventTypes("invalidType"),
					withCredentials(saKey),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.eventTypes: Unsupported value: "invalidType"`))
			})
		})
	})
})

// createSource creates a GoogleCloudStorage object initialized with the given options.
func createSource(srcClient dynamic.ResourceInterface, namespace, namePrefix string,
	sink *duckv1.Destination, opts ...sourceOption) (*unstructured.Unstructured, error) {

	src := &unstructured.Unstructured{}
	src.SetAPIVersion(sourceAPIVersion.String())
	src.SetKind(sourceKind)
	src.SetNamespace(namespace)
	src.SetGenerateName(namePrefix)

	if err := unstructured.SetNestedMap(src.Object, ducktypes.DestinationToMap(sink), "spec", "sink"); err != nil {
		framework.FailfWithOffset(2, "Failed to set spec.sink field: %s", err)
	}

	for _, opt := range opts {
		opt(src)
	}

	return srcClient.Create(context.Background(), src, metav1.CreateOptions{})
}

type sourceOption func(*unstructured.Unstructured)

func withBucket(bucket string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, bucket, "spec", "bucket"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.bucket field: %s", err)
		}
	}
}

func withProject(project string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, project, "spec", "pubsub", "project"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.pubsub.project field: %s", err)
		}
	}
}

func withEventTypes(eventType string) sourceOption {
	return func(src *unstructured.Unstructured) {
		eventTypes, _, err := unstructured.NestedStringSlice(src.Object, "spec", "eventTypes")
		if err != nil {
			framework.FailfWithOffset(3, "Error reading spec.eventTypes field: %s", err)
		}

		eventTypes = append(eventTypes, eventType)

		if err := unstructured.SetNestedStringSlice(src.Object, eventTypes, "spec", "eventTypes"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.eventTypes field: %s", err)
		}
	}
}

func withCredentials(creds string) sourceOption {
	c := map[string]interface{}{"value": creds}

	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, c, "spec", "serviceAccountKey"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.serviceAccountKey field: %s", err)
		}
	}
}

// readReceivedEvents returns a function that reads CloudEvents received by the
// event-display application and stores the result as the value of the given
// `receivedEvents` variable.
// The returned function signature satisfies the contract expected by
// gomega.Eventually: no argument and one or more return values.
func readReceivedEvents(c clientset.Interface, namespace, eventDisplayDeplName string,
	receivedEvents *[]cloudevents.Event) func() []cloudevents.Event {

	return func() []cloudevents.Event {
		ev := bridges.ReceivedEventDisplayEvents(
			apps.GetLogs(c, namespace, eventDisplayDeplName),
		)
		*receivedEvents = ev
		return ev
	}
}
