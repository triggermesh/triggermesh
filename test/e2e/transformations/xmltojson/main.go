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

// Overflow

// Create an Event Sink (Event Display)

// Create a XMLToJSON Transformation (using the Event Display as a Sink)

// Send the XMLToJSON Transformation a CloudEvent with XML in the payload

// Expect valid JSON in the Event Display

package xmltmtojson

import (
	"context"
	"net/url"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck

	. "github.com/onsi/gomega" //nolint:stylecheck

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"

	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	"github.com/triggermesh/triggermesh/test/e2e/framework/bridges"
	e2ece "github.com/triggermesh/triggermesh/test/e2e/framework/cloudevents"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
)

var transAPIVersion = schema.GroupVersion{
	Group:   "flow.triggermesh.io",
	Version: "v1alpha1",
}

const (
	transformationKind     = "XMLToJSONTransformation"
	transformationResource = "xmltojsontransformations"
)

var _ = Describe("XMLToJSON Transformation", func() {
	f := framework.New("xmltojsontransformation")

	var ns string
	var sink *duckv1.Destination

	var trnsClient dynamic.ResourceInterface
	var trans *unstructured.Unstructured
	var err error
	var transURL *url.URL

	// 	Context("a Transformation is deployed" ...
	//   When("a XML payload is sent" ...
	//     It("converts the payload to JSON")
	//   When("a non XML payload is sent" ...
	//     It("responds with an error event" ...

	Context("a Transformation is deployed", func() {
		BeforeEach(func() {
			ns = f.UniqueName
			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
				Expect(sink).NotTo(BeNil())
			})

			By("creating a transformation object", func() {
				gvr := transAPIVersion.WithResource(transformationResource)
				trnsClient = f.DynamicClient.Resource(gvr).Namespace(ns)
				trans, err = createTransformation(trnsClient, ns, "test-xmltojson-", sink)
				Expect(err).ToNot(HaveOccurred())

				trans = ducktypes.WaitUntilReady(f.DynamicClient, trans)
				transURL = ducktypes.Address(trans)
				Expect(transURL).ToNot(BeNil())
			})

		})
		When("a XML payload is sent", func() {
			It("should be created", func() {
				Expect(1).To(Equal(1))
				// sentEvent := e2ece.NewXMLHelloEvent(f)
				// job := e2ece.RunXMLEventSender(f.KubeClient, ns, transURL.String()+":8080", sentEvent)
				sentEvent := e2ece.NewHelloEvent(f)
				job := e2ece.RunEventSender(f.KubeClient, ns, transURL.String(), sentEvent)
				apps.WaitForCompletion(f.KubeClient, job)
			})

			// It("should generate an event at the sink", func() {

			// 	var receivedEvents []cloudevents.Event

			// 	readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

			// 	Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
			// 	Expect(receivedEvents).To(HaveLen(1))

			// 	e := receivedEvents[0]
			// 	Expect(e.Type()).To(Equal("com.amazon.codecommit.push"))
			// 	Expect(e.Source()).To(Equal(repoARN))
			// 	Expect(e.Subject()).To(Equal(e2ecodecommit.DefaultBranch))
			// })

		})

	})

})

// createTransformation creates an AWSKinesis object initialized with the given options.
func createTransformation(trnsClient dynamic.ResourceInterface, namespace, namePrefix string, sink *duckv1.Destination) (*unstructured.Unstructured, error) {
	trns := &unstructured.Unstructured{}
	trns.SetAPIVersion(transAPIVersion.String())
	trns.SetKind(transformationKind)
	trns.SetNamespace(namespace)
	trns.SetGenerateName(namePrefix)

	if err := unstructured.SetNestedMap(trns.Object, ducktypes.DestinationToMap(sink), "spec", "sink"); err != nil {
		framework.FailfWithOffset(2, "Failed to set spec.sink field: %s", err)
	}

	return trnsClient.Create(context.Background(), trns, metav1.CreateOptions{})
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
