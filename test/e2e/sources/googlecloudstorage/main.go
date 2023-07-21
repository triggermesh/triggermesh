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

package googlecloudstorage

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

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
)

var _ = Describe("Google Cloud Storage source", func() {
	// NOTE: bucket names aren't allowed to contain "goog"
	f := framework.New("gcloudstoragesource")

	var ns string

	var srcClient dynamic.ResourceInterface

	var bucketID string
	var gcloudProject string
	var serviceaccountKey string

	var sink *duckv1.Destination

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := sourceAPIVersion.WithResource(sourceResource)
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source subscribes to a bucket's change notifications", func() {
		var storageClient *storage.Client

		BeforeEach(func() {
			serviceaccountKey = e2egcloud.ServiceAccountKeyFromEnv()
			gcloudProject = e2egcloud.ProjectNameFromEnv()

			var err error

			storageClient, err = storage.NewClient(context.Background(), option.WithCredentialsJSON([]byte(serviceaccountKey)))
			Expect(err).ToNot(HaveOccurred())

			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
			})

			By("creating a bucket", func() {
				bucketID = e2estorage.CreateBucket(storageClient, gcloudProject, f)

				DeferCleanup(func() {
					By("deleting storage bucket "+bucketID, func() {
						e2estorage.DeleteBucket(storageClient, bucketID)
					})
				})
			})

			By("creating a GoogleCloudStorageSource object", func() {
				src, err := createSource(srcClient, ns, "test-", sink,
					withBucket(bucketID),
					withProject(gcloudProject),
					withEventType("OBJECT_FINALIZE"),
					withServiceAccountKey(serviceaccountKey),
				)
				Expect(err).ToNot(HaveOccurred())

				ducktypes.WaitUntilReady(f.DynamicClient, src)
			})
		})

		When("a new object is created", func() {
			var objectName string

			BeforeEach(func() {
				objectName = e2estorage.CreateObject(storageClient, bucketID, f)

				DeferCleanup(func() {
					By("deleting object "+objectName+" from storage bucket "+bucketID, func() {
						e2estorage.DeleteObject(storageClient, bucketID, objectName)
					})
				})
			})

			Specify("the source generates an event", func() {
				const receiveTimeout = 150 * time.Second
				const pollInterval = 500 * time.Millisecond

				var receivedEvents []cloudevents.Event

				readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

				Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
				Expect(receivedEvents).To(HaveLen(1))

				e := receivedEvents[0]

				Expect(e.Type()).To(Equal("com.google.cloud.storage.objectfinalize"))
				Expect(e.Source()).To(Equal("gs://" + bucketID))

				var msg = struct {
					Attributes map[string]string
				}{}
				Expect(e.DataAs(&msg)).ToNot(HaveOccurred())

				Expect(msg.Attributes["bucketId"]).To(Equal(bucketID))
				Expect(msg.Attributes["eventType"]).To(Equal("OBJECT_FINALIZE"))
				Expect(msg.Attributes["objectId"]).To(Equal(objectName))
			})
		})
	})

	When("a client creates a source object with invalid specs", func() {

		// Those tests do not require a real bucket or sink
		BeforeEach(func() {
			bucketID = "fake-bucket"
			gcloudProject = "fake-project"

			serviceaccountKey = "fake-creds"

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
				_, err := createSource(srcClient, ns, "test-invalid-bucket-", sink,
					withBucket("Invalid_Bucket_Name"),
					withProject(gcloudProject),
					withServiceAccountKey(serviceaccountKey),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.bucket: Invalid value: "))
			})

			By("setting an invalid event type", func() {
				_, err := createSource(srcClient, ns, "test-invalid-eventtype-", sink,
					withBucket(bucketID),
					withProject(gcloudProject),
					withEventType("invalid_type"),
					withServiceAccountKey(serviceaccountKey),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.eventTypes[0]: Unsupported value: "invalid_type"`))
			})

			By("setting an invalid project", func() {
				_, err := createSource(srcClient, ns, "test-invalid-project-", sink,
					withBucket(bucketID),
					withProject("invalid_project"),
					withServiceAccountKey(serviceaccountKey),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.pubsub.project: Invalid value: "))
			})

			By("omitting the project", func() {
				_, err := createSource(srcClient, ns, "test-noproject-", sink,
					withBucket(bucketID),
					withServiceAccountKey(serviceaccountKey),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.pubsub: Required value"))
			})

			By("setting empty credentials", func() {
				_, err := createSource(srcClient, ns, "test-nocreds-", sink,
					withBucket(bucketID),
					withProject(gcloudProject),
					withServiceAccountKey(""),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`"spec.auth.serviceAccountKey" must validate one and only one schema (oneOf).`))
			})
		})
	})
})

// createSource creates a GoogleCloudStorageSource object initialized with the given options.
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

func withEventType(eventType string) sourceOption {
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

func withServiceAccountKey(key string) sourceOption {
	svcAccKeyMap := make(map[string]interface{})
	if key != "" {
		svcAccKeyMap = map[string]interface{}{"value": key}
	}

	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedMap(src.Object, svcAccKeyMap, "spec", "auth", "serviceAccountKey"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.auth.serviceAccountKey field: %s", err)
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
