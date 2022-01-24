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

package googlecloudauditlogs

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

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	"github.com/triggermesh/triggermesh/test/e2e/framework/bridges"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
	e2egcloud "github.com/triggermesh/triggermesh/test/e2e/framework/gcloud"
	e2epubsub "github.com/triggermesh/triggermesh/test/e2e/framework/gcloud/pubsub"
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
	sourceKind     = "GoogleCloudAuditLogsSource"
	sourceResource = "googlecloudauditlogssources"
)

var _ = Describe("Google Cloud Audit Logs source", func() {
	f := framework.New("googlecloudauditlogssource")

	var ns string

	var srcClient dynamic.ResourceInterface

	var gcloudProject string
	var serviceaccountKey string

	var sink *duckv1.Destination

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := sourceAPIVersion.WithResource(sourceResource)
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source subscribes to audit logs for the Cloud Pub/Sub service", func() {
		var pubsubClient *pubsub.Client
		var err error

		const serviceNamePubSub = "pubsub.googleapis.com"
		const methodNamePubSubCreateTopic = "google.pubsub.v1.Publisher.CreateTopic"

		BeforeEach(func() {
			serviceaccountKey = e2egcloud.ServiceAccountKeyFromEnv()
			gcloudProject = e2egcloud.ProjectNameFromEnv()

			pubsubClient, err = pubsub.NewClient(context.Background(), gcloudProject, option.WithCredentialsJSON([]byte(serviceaccountKey)))
			Expect(err).ToNot(HaveOccurred())

			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
			})

			By("creating a GoogleCloudAuditLogsSource object", func() {
				src, err := createSource(srcClient, ns, "test-", sink,
					withServiceName(serviceNamePubSub),
					withMethodName(methodNamePubSubCreateTopic),
					withProject(gcloudProject),
					withServiceAccountKey(serviceaccountKey),
				)
				Expect(err).ToNot(HaveOccurred())

				ducktypes.WaitUntilReady(f.DynamicClient, src)

				// FIXME: We observed that audit logs generated shortly after the source reports "Ready"
				// weren't sent to the Pub/Sub topic observed by the receive adapter. It is likely that
				// the Audit Logs Router Sink reconciled by the source starts routing audit logs after a
				// delay (10-150s), so we virtually delay the next test steps here as well.
				// https://github.com/triggermesh/triggermesh/issues/469
				time.Sleep(150 * time.Second)
			})
		})

		When("a Pub/Sub topic is created", func() {

			BeforeEach(func() {
				By("creating a Pub/Sub topic", func() {
					topicID := e2epubsub.CreateTopic(pubsubClient, f).ID()

					DeferCleanup(func() {
						By("deleting the Pub/Sub topic "+topicID, func() {
							e2epubsub.DeleteTopic(pubsubClient, topicID)
						})
					})
				})
			})

			Specify("the source generates an event", func() {
				const receiveTimeout = 15 * time.Second
				const pollInterval = 500 * time.Millisecond

				var receivedEvents []cloudevents.Event

				readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

				Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())

				e := receivedEvents[0]

				Expect(e.Type()).To(Equal("com.google.cloud.auditlogs.notification"))
				Expect(e.Source()).To(Equal("pubsub.googleapis.com"))
			})
		})
	})

	When("a client creates a source object with invalid specs", func() {
		const serviceName = "foo.example.com"
		const methodName = "foo.v0.DoSomething"

		// Those tests do not require a real project or sink
		BeforeEach(func() {
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

			By("setting an invalid service name", func() {
				const invalidServiceName = "Foo.example.com"

				_, err := createSource(srcClient, ns, "test-invalid-servicename-", sink,
					withServiceName(invalidServiceName),
					withMethodName(methodName),
					withProject(gcloudProject),
					withServiceAccountKey(serviceaccountKey),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.serviceName: Invalid value: "))
			})

			By("setting an invalid method name", func() {
				const invalidMethodName = "foo.v0."

				_, err := createSource(srcClient, ns, "test-invalid-methodname-", sink,
					withServiceName(serviceName),
					withMethodName(invalidMethodName),
					withProject(gcloudProject),
					withServiceAccountKey(serviceaccountKey),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.methodName: Invalid value: "))
			})

			By("setting an invalid project", func() {
				_, err := createSource(srcClient, ns, "test-invalid-project-", sink,
					withServiceName(serviceName),
					withMethodName(methodName),
					withProject("invalid_project"),
					withServiceAccountKey(serviceaccountKey),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.pubsub.project: Invalid value: "))
			})

			By("omitting the project", func() {
				_, err := createSource(srcClient, ns, "test-noproject-", sink,
					withServiceName(serviceName),
					withMethodName(methodName),
					withServiceAccountKey(serviceaccountKey),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.pubsub: Required value"))
			})

			By("setting empty credentials", func() {
				_, err := createSource(srcClient, ns, "test-nocreds-", sink,
					withServiceName(serviceName),
					withMethodName(methodName),
					withProject(gcloudProject),
					withServiceAccountKey(""),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`"spec.serviceAccountKey" must validate one and only one schema (oneOf).`))
			})
		})
	})
})

// createSource creates an GoogleCloudAuditLogsSource object initialized with the given options.
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

func withServiceName(serviceName string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, serviceName, "spec", "serviceName"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.serviceName field: %s", err)
		}
	}
}

func withMethodName(methodName string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, methodName, "spec", "methodName"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.methodName field: %s", err)
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

func withServiceAccountKey(key string) sourceOption {
	svcAccKeyMap := make(map[string]interface{})
	if key != "" {
		svcAccKeyMap = map[string]interface{}{"value": key}
	}

	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedMap(src.Object, svcAccKeyMap, "spec", "serviceAccountKey"); err != nil {
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
