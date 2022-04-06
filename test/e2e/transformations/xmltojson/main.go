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

package xmltmtojson

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
	e2ece "github.com/triggermesh/triggermesh/test/e2e/framework/cloudevents"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
)

var flowAPIVersion = schema.GroupVersion{
	Group:   "flow.triggermesh.io",
	Version: "v1alpha1",
}

const (
	transformationKind     = "XMLToJSONTransformation"
	transformationResource = "xmltojsontransformations"
)

const (
	xmlPayload = `<note>` +
		`<to>Tove</to>` +
		`<from>Jani</from>` +
		`<heading>Reminder</heading>` +
		`<body>Dont forget me this weekend</body>` +
		`</note>`

	expectJSONData = `{` +
		`"note":{` +
		`"body":"Dont forget me this weekend",` +
		`"from":"Jani",` +
		`"heading":"Reminder",` +
		`"to":"Tove"` +
		`}` +
		`}`
)

var _ = Describe("XMLToJSON Transformation", func() {
	f := framework.New("xmltojsontransformation")

	var ns string

	var trnsClient dynamic.ResourceInterface

	var transURL *url.URL

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := flowAPIVersion.WithResource(transformationResource)
		trnsClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a Transformation is deployed with a sink set as destination", func() {
		var sink *duckv1.Destination

		BeforeEach(func() {

			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
			})

			By("creating a XMLToJSONTransformation object", func() {
				trans, err := createTransformation(trnsClient, ns, "test-sink-",
					withSink(sink),
				)
				Expect(err).ToNot(HaveOccurred())

				trans = ducktypes.WaitUntilReady(f.DynamicClient, trans)

				transURL = ducktypes.Address(trans)
				Expect(transURL).ToNot(BeNil())
			})

		})

		When("a XML payload is sent", func() {
			var sentEvent *cloudevents.Event

			BeforeEach(func() {
				sentEvent = newXMLHelloEvent(f)

				job := e2ece.RunEventSender(f.KubeClient, ns, transURL.String(), sentEvent)
				apps.WaitForCompletion(f.KubeClient, job)
			})

			It("transforms the payload to JSON and sends it to the sink", func() {
				const receiveTimeout = 10 * time.Second
				const pollInterval = 500 * time.Millisecond

				var receivedEvents []cloudevents.Event

				readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

				Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
				Expect(receivedEvents).To(HaveLen(1))

				gotEvent := receivedEvents[0]

				Expect(gotEvent.Data()).To(Equal([]byte(expectJSONData)))
				Expect(gotEvent.DataContentType()).To(Equal(cloudevents.ApplicationJSON))

				Expect(gotEvent.ID()).To(Equal(sentEvent.ID()))
				Expect(gotEvent.Type()).To(Equal(sentEvent.Type()))
				Expect(gotEvent.Source()).To(Equal(sentEvent.Source()))
				Expect(gotEvent.Extensions()[e2ece.E2ECeExtension]).
					To(Equal(sentEvent.Extensions()[e2ece.E2ECeExtension]))
			})
		})
	})

	Context("a Transformation is deployed without a sink", func() {
		var entrypointURL *url.URL
		var repliesDisplayDeplName string

		BeforeEach(func() {
			var transDst *duckv1.Destination

			By("creating a XMLToJSONTransformation object", func() {
				trans, err := createTransformation(trnsClient, ns, "test-nosink-")
				Expect(err).ToNot(HaveOccurred())

				trans = ducktypes.WaitUntilReady(f.DynamicClient, trans)

				transDst = &duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: trans.GetAPIVersion(),
						Kind:       trans.GetKind(),
						Name:       trans.GetName(),
					},
				}
			})

			By("creating a response intercepter", func() {
				entrypointURL, repliesDisplayDeplName = bridges.SetupSubscriberWithReplyTo(
					f.KubeClient, f.DynamicClient, ns, transDst)
			})
		})

		When("a XML payload is sent", func() {
			var sentEvent *cloudevents.Event

			BeforeEach(func() {
				sentEvent = newXMLHelloEvent(f)

				job := e2ece.RunEventSender(f.KubeClient, ns, entrypointURL.String(), sentEvent)
				apps.WaitForCompletion(f.KubeClient, job)
			})

			It("transforms the payload to JSON and replies with the transformed data", func() {
				const receiveTimeout = 10 * time.Second
				const pollInterval = 500 * time.Millisecond

				var receivedEvents []cloudevents.Event

				readReceivedEvents := readReceivedEvents(f.KubeClient, ns, repliesDisplayDeplName, &receivedEvents)

				Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
				Expect(receivedEvents).To(HaveLen(1))

				gotEvent := receivedEvents[0]

				Expect(gotEvent.Data()).To(Equal([]byte(expectJSONData)))
				Expect(gotEvent.DataContentType()).To(Equal(cloudevents.ApplicationJSON))

				Expect(gotEvent.ID()).To(Equal(sentEvent.ID()))
				Expect(gotEvent.Type()).To(Equal(sentEvent.Type()))
				Expect(gotEvent.Source()).To(Equal(sentEvent.Source()))
				Expect(gotEvent.Extensions()[e2ece.E2ECeExtension]).
					To(Equal(sentEvent.Extensions()[e2ece.E2ECeExtension]))
			})
		})
	})
})

// newXMLHelloEvent generates a CloudEvent with dummy values and an XML data payload.
func newXMLHelloEvent(f *framework.Framework) *cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetID("0000")
	event.SetType("e2e.test")
	event.SetSource("e2e.triggermesh")
	event.SetExtension(e2ece.E2ECeExtension, f.UniqueName)
	if err := event.SetData(cloudevents.ApplicationXML, []byte(xmlPayload)); err != nil {
		framework.FailfWithOffset(2, "Error setting event data: %s", err)
	}
	event.DataBase64 = false

	return &event
}

// createTransformation creates an XMLToJSONTransformation object initialized with the given options.
func createTransformation(trnsClient dynamic.ResourceInterface, namespace, namePrefix string,
	opts ...transformationOption) (*unstructured.Unstructured, error) {

	trns := &unstructured.Unstructured{}
	trns.SetAPIVersion(flowAPIVersion.String())
	trns.SetKind(transformationKind)
	trns.SetNamespace(namespace)
	trns.SetGenerateName(namePrefix)

	for _, opt := range opts {
		opt(trns)
	}

	return trnsClient.Create(context.Background(), trns, metav1.CreateOptions{})
}

type transformationOption func(*unstructured.Unstructured)

func withSink(sink *duckv1.Destination) transformationOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedMap(src.Object, ducktypes.DestinationToMap(sink), "spec", "sink"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.sink field: %s", err)
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
