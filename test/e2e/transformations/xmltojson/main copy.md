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



import (
	"context"
	"net/url"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"

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
	// Context("test", func() {
	f := framework.New("xmltojsontransformation")

	var ns string
	var trnsClient dynamic.ResourceInterface

	// When("test", func() {
	// 	It("should", func() {
	// 		Expect(nil).To(nil)
	// 	})
	// })

	// BeforeEach(func() {
	ns = f.UniqueName
	var transURL *url.URL

	gvr := transAPIVersion.WithResource(transformationResource)
	trnsClient = f.DynamicClient.Resource(gvr).Namespace(ns)

	sink := bridges.CreateEventDisplaySink(f.KubeClient, ns)

	trans, err := createTransformation(trnsClient, ns, "test-xmltojson-", sink.URI.String())
	Expect(err).To(nil)

	By("receving an XML cloudevent", func() {
		sentEvent := e2ece.NewXMLHelloEvent(f)

		trans = ducktypes.WaitUntilReady(f.DynamicClient, trans)

		transURL = ducktypes.Address(trans)
		Expect(transURL).ToNot(BeNil())

		job := e2ece.RunXMLEventSender(f.KubeClient, ns, transURL.String(), sentEvent)
		apps.WaitForCompletion(f.KubeClient, job)

	})

	// malnamed
	By("sink reciving events", func() {
		const receiveTimeout = 15 * time.Second
		const pollInterval = 500 * time.Millisecond

		var receivedEvents []cloudevents.Event

		readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

		Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
		Expect(receivedEvents).To(HaveLen(1))
	})

})

// })

// })

// createTransformation creates an AWSKinesis object initialized with the given options.
func createTransformation(trnsClient dynamic.ResourceInterface, namespace, namePrefix, sink string) (*unstructured.Unstructured, error) {

	trns := &unstructured.Unstructured{}
	trns.SetAPIVersion(transAPIVersion.String())
	trns.SetKind(transformationKind)
	trns.SetNamespace(namespace)
	trns.SetGenerateName(namePrefix)
	trns.Object["spec"] = map[string]interface{}{"K_SINK": sink}

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
