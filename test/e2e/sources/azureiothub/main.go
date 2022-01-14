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

package azureiothub

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

	"github.com/triggermesh/triggermesh/test/e2e/framework/azure"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	"github.com/triggermesh/triggermesh/test/e2e/framework/bridges"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
)

/*
  This test requires:
  - Azure Service Principal Credentials with the Azure Event Hubs Data Owner role assigned at the subscription level

  The following environment variables _MUST_ be set:
  - AZURE_SUBSCRIPTION_ID - Common subscription for the test to run against
  - AZURE_TENANT_ID - Azure tenant to create the resources against
  - AZURE_CLIENT_ID - The Azure ServicePrincipal Client ID
  - AZURE_CLIENT_SECRET - The Azure ServicePrincipal Client Secret
  - AZURE_REGION - Define the Azure region to run the test (default uswest2)

  These will be done by the e2e test:
  - Create an Azure IoT Hub and device
  - Obtain the iothubowner credentials and use it for creating the AzureIOTHubService
  - Instantiate a pseudodevice based on the device registerd, and use that to produce
    telementry data.
*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "AzureIOTHubSource"
	sourceResource = "azureiothubsource"
)

var _ = Describe("Azure IOT Hub Source", func() {
	ctx := context.Background()
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	region := os.Getenv("AZURE_REGION")

	if region == "" {
		region = "westus2"
	}

	f := framework.New(sourceResource)

	var ns string
	var srcClient dynamic.ResourceInterface
	var sink *duckv1.Destination

	var rg armresources.ResourceGroup
	var iotHubAddress string
	var deviceKey string

	BeforeEach(func() {
		ns = f.UniqueName
		gvr := sourceAPIVersion.WithResource(sourceResource + "s")
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source watches an IOTHub for a device message", func() {
		var err error // stubbed

		When("an event flows", func() {
			BeforeEach(func() {
				rg = azure.CreateResourceGroup(ctx, subscriptionID, ns, region)
				deviceKey, iotHubAddress = azure.CreateIOTHubComponents(ctx, subscriptionID, *rg.Name, region, ns)
			})

			var src *unstructured.Unstructured

			BeforeEach(func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)

				src, err = createSource(srcClient, ns, "test-", sink,
					withSASToken(iotHubAddress),
				)

				Expect(err).ToNot(HaveOccurred())
				ducktypes.WaitUntilReady(f.DynamicClient, src)

				// FIXME(antoineco): because it doesn't have any startup probe, the adapter becomes
				// Ready before it has established a connection with the IoT Hub. For this reason,
				// sending a message immediately after observing the Ready condition occasionally causes
				// a race.
				time.Sleep(10 * time.Second)
			})
			It("should create an azure iothub source", func() {
				By("creating a message sent from a device", func() {
					CreateMsg(ns, "testdev", deviceKey)
				})

				By("verifying the event was sent to a newly created iothub", func() {
					const receiveTimeout = 30 * time.Second
					const pollInterval = 500 * time.Millisecond

					var receivedEvents []cloudevents.Event

					readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

					Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
					Expect(receivedEvents).To(HaveLen(1))

					e := receivedEvents[0]

					Expect(e.Type()).To(Equal("com.microsoft.azure.iothub.message"))
					Expect(e.Source()).To(Equal(ns + ".azure-devices.net"))

					data := make(map[string]interface{})
					_ = json.Unmarshal(e.Data(), &data)

					Expect(data["MessageSource"]).To(Equal("Telemetry"))
					payload, _ := base64.StdEncoding.DecodeString(data["Payload"].(string))
					testData := make(map[string]interface{})
					_ = json.Unmarshal(payload, &testData)
					Expect(testData["id"]).To(Equal("1"))
					Expect(testData["data"]).To(Equal("a test payload"))
				})
			})
			AfterEach(func() {
				_ = azure.DeleteResourceGroup(ctx, subscriptionID, *rg.Name)
			})
		})
	})

	When("a client creates a source object with invalid specs", func() {
		// Those tests do not require a real sink
		BeforeEach(func() {
			sink = &duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: "fake/v1",
					Kind:       "Fake",
					Name:       "fake",
				},
			}
		})

		Specify("the API server rejects the creation of that object", func() {
			By("omitting credentials", func() {
				_, err := createSource(srcClient, ns, "test-empty-credentials", sink)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.auth: Required value`))
			})
		})
	})
})

type sourceOption func(*unstructured.Unstructured)

// createSource creates an AzureIOTHubSource object initialized with the test parameters
func createSource(srcClient dynamic.ResourceInterface, namespace, namePrefix string,
	sink *duckv1.Destination, opts ...sourceOption) (*unstructured.Unstructured, error) {
	src := &unstructured.Unstructured{}
	src.SetAPIVersion(sourceAPIVersion.String())
	src.SetKind(sourceKind)
	src.SetNamespace(namespace)
	src.SetGenerateName(namePrefix)

	// Set spec parameters

	if err := unstructured.SetNestedMap(src.Object, ducktypes.DestinationToMap(sink), "spec", "sink"); err != nil {
		framework.FailfWithOffset(2, "Failed to set spec.sink field: %s", err)
	}

	for _, opt := range opts {
		opt(src)
	}

	return srcClient.Create(context.Background(), src, metav1.CreateOptions{})
}

// Define the creation parameters to pass along

// withSASToken will create the secret and service principal based on the azure environment variables
func withSASToken(sasToken string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, sasToken, "spec", "auth", "sasToken", "connectionString", "value"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.auth.sasToken.connectionString.value: %s", err)
		}
	}
}

// readReceivedEvents returns a function that reads CloudEvents received by the
// event-display application and stores the result as the value of the given
// `receivedEvents` variable.
// The returned function signature satisfies the contract expected by
// gomega.Eventually: no argument and one or more return values.
func readReceivedEvents(c clientset.Interface, namespace, eventDisplayName string,
	receivedEvents *[]cloudevents.Event) func() []cloudevents.Event {

	return func() []cloudevents.Event {
		ev := bridges.ReceivedEventDisplayEvents(
			apps.GetLogs(c, namespace, eventDisplayName),
		)
		*receivedEvents = ev
		return ev
	}
}

// CreateMsg generates a new test message as if it originated from the device.
// NOTE: The Azure IOTHub SDK does not exist for Go, so the REST API must be used.
func CreateMsg(hubName, deviceName, deviceKey string) {
	baseURL := fmt.Sprintf("%s.azure-devices.net/devices/%s", hubName, deviceName)
	payload := "{ \"id\": \"1\", \"data\":\"a test payload\"}"
	nr := strings.NewReader(payload)

	req, _ := http.NewRequest(http.MethodPost, "https://"+baseURL+"/messages/events?api-version=2020-03-13", nr)
	req.Header.Add("Authorization", azure.CreateSaSToken(baseURL, deviceName, deviceKey, false))
	req.Header.Add("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)

	if err != nil || resp.StatusCode != http.StatusNoContent {
		framework.FailfWithOffset(2, "Unable to send message: %s %s", err, buf.String())
	}
}
