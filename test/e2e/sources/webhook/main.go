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

package webhook

import (
	"context"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"

	duckv1 "knative.dev/pkg/apis/duck/v1"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	"github.com/triggermesh/triggermesh/test/e2e/framework/bridges"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
	"github.com/triggermesh/triggermesh/test/e2e/framework/http"
)

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "WebhookSource"
	sourceResource = "webhooksources"
)

var _ = Describe("Webhook source", func() {
	f := framework.New("webhooksource")

	var ns string

	var srcClient dynamic.ResourceInterface

	var eventType string
	var eventSource string
	var sink *duckv1.Destination

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := sourceAPIVersion.WithResource(sourceResource)
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source receives an HTTP request", func() {
		var srcURL *url.URL

		// sample payload struct and instance to be
		// sent at the WebhookSource URL
		type Payload struct {
			Message string
		}
		testPayload := Payload{"test"}

		BeforeEach(func() {
			eventType = "test.event.type"
			eventSource = "test.event.source"

			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
			})

			By("creating an WebhookSource object", func() {
				src, err := createSource(srcClient, ns, "test-", sink,
					withEventType(eventType),
					withEventSource(eventSource),
				)
				Expect(err).ToNot(HaveOccurred())

				src = ducktypes.WaitUntilReady(f.DynamicClient, src)

				srcURL = ducktypes.Address(src)
				Expect(srcURL).ToNot(BeNil())
			})
		})

		When("an HTTP request is received", func() {
			BeforeEach(func() {
				http.PostJSONRequestWithRetries(5*time.Second, 1*time.Minute, srcURL.String(), testPayload)
			})

			Specify("the source generates an event", func() {
				const receiveTimeout = 15 * time.Second
				const pollInterval = 500 * time.Millisecond

				var receivedEvents []cloudevents.Event

				readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

				Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
				Expect(receivedEvents).To(HaveLen(1))

				e := receivedEvents[0]

				Expect(e.Type()).To(Equal(eventType))
				Expect(e.Source()).To(Equal(eventSource))

				tp := Payload{}
				err := e.DataAs(&tp)
				Expect(err).ToNot(HaveOccurred())
				Expect(tp).To(Equal(testPayload))
			})
		})
	})

	When("a client creates a source object with invalid specs", func() {

		// Those tests do not require a real repository or sink
		BeforeEach(func() {
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

			By("not setting an event type", func() {
				_, err := createSource(srcClient, ns, "test-", sink)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.eventType: Required value"))
			})
		})
	})
})

// createSource creates a WebhookSource object initialized with the given options.
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

func withEventType(eventType string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, eventType, "spec", "eventType"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.eventType field: %s", err)
		}
	}
}

func withEventSource(eventSource string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, eventSource, "spec", "eventSource"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.eventSource field: %s", err)
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
