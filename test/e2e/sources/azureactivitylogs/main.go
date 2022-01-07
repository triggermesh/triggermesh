/*
Copyright (c) 2022 TriggerMesh Inc.

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

package azureactivitylogs

import (
	"context"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

	cloudevents "github.com/cloudevents/sdk-go/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	"github.com/triggermesh/triggermesh/test/e2e/framework/azure"
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
  - Create an Azure Resource Group, EventHubs Namespace, and EventHub
  - Send an event from the Azure EventHub into the TriggerMesh source

*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "AzureActivityLogsSource"
	sourceResource = "azureactivitylogssource"
)

/*
 Basic flow will resemble:
 * Create a resource group to contain our eventhub
 * Ensure our service principal can read/write from the eventhub
 * Instantiate the AzureActivityLogsSource
 * Create a resource group and watch the event flow in
*/

var _ = Describe("Azure Activity Logs", func() {
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

	BeforeEach(func() {
		ns = f.UniqueName
		gvr := sourceAPIVersion.WithResource(sourceResource + "s")
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source watches an EventHub publishing Activity Log data", func() {
		var err error // stubbed
		var testRG armresources.ResourceGroup

		When("an event flows", func() {
			BeforeEach(func() {
				rg = azure.CreateResourceGroup(ctx, subscriptionID, ns, region)
				_ = azure.CreateEventHubComponents(ctx, subscriptionID, ns, region, *rg.Name)
			})

			It("should create an azure eventhub", func() {
				By("creating an event sink", func() {
					sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
				})

				By("creating a sample resource group to produce activity", func() {
					testRG = azure.CreateResourceGroup(ctx, subscriptionID, *rg.Name+"-testrg", region)
				})

				var src *unstructured.Unstructured
				By("creating the azureactivitylog source", func() {
					src, err = createSource(srcClient, ns, "test-", sink,
						withServicePrincipal(),
						withSubscriptionID(subscriptionID),
						withActivityCategories([]string{"Administrative", "Policy", "Security"}),
						withEventHubNS(createEventHubNS(subscriptionID, ns)),
						withEventHubName(ns),
					)

					Expect(err).ToNot(HaveOccurred())

					ducktypes.WaitUntilReady(f.DynamicClient, src)
				})

				By("verifying the event was sent by deleting a resource", func() {
					deleteFuture := azure.DeleteResourceGroup(ctx, subscriptionID, *testRG.Name)
					azure.WaitForFutureDeletion(ctx, subscriptionID, deleteFuture)

					const receiveTimeout = 900 * time.Second // It can take up to 15 minutes for an event to appear
					const pollInterval = 500 * time.Millisecond

					var receivedEvents []cloudevents.Event

					readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

					Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
					Expect(receivedEvents).ToNot(BeEmpty()) // In some cases will receive either 1 or 2 events
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

			By("setting an invalid subscriptionID", func() {
				fakeSubID := "I'm a fake subscription"

				_, err := createSource(srcClient, ns, "test-invalid-sub-id", sink,
					withServicePrincipal(),
					withSubscriptionID(fakeSubID),
					withActivityCategories([]string{"Administrative", "Policy", "Security"}),
					withEventHubNS(createEventHubNS(subscriptionID, ns)),
					withEventHubName(ns),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.subscriptionID: Invalid value: "))
			})

			By("omitting credentials", func() {
				_, err := createSource(srcClient, ns, "test-nocreds-", sink,
					withSubscriptionID(subscriptionID),
					withActivityCategories([]string{"Administrative", "Policy", "Security"}),
					withEventHubNS(createEventHubNS(subscriptionID, ns)),
					withEventHubName(ns),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.auth: Required value`))
			})

			By("omitting destination", func() {
				_, err := createSource(srcClient, ns, "test-no-eventhubs-", sink,
					withServicePrincipal(),
					withSubscriptionID(subscriptionID),
					withActivityCategories([]string{"Administrative", "Policy", "Security"}),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.destination: Required value`))
			})

			By("setting invalid eventhub namespace", func() {
				fakeEventhubNamespace := "I'm a fake eventhub namespace"
				_, err := createSource(srcClient, ns, "test-invalid-eventhub-ns", sink,
					withServicePrincipal(),
					withSubscriptionID(subscriptionID),
					withActivityCategories([]string{"Administrative", "Policy", "Security"}),
					withEventHubNS(fakeEventhubNamespace),
					withEventHubName(ns),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.destination.eventHubs.namespaceID: Invalid value: "`))
			})

			By("setting invalid eventhub name", func() {
				fakeName := "I'm a fake name"
				_, err := createSource(srcClient, ns, "test-invalid-eventhub-name", sink,
					withServicePrincipal(),
					withSubscriptionID(subscriptionID),
					withEventHubNS(createEventHubNS(subscriptionID, ns)),
					withEventHubName(fakeName),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.destination.eventHubs.hubName: Invalid value: "`))
			})
		})
	})
})

type sourceOption func(*unstructured.Unstructured)

// createSource creates an AzureEventHubSource object initialized with the test parameters
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

func withEventHubNS(ns string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, ns, "spec", "destination", "eventHubs", "namespaceID"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.destination.eventHubs.namespaceID: %s", err)
		}
	}
}

func withEventHubName(name string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, name, "spec", "destination", "eventHubs", "hubName"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.destination.eventHubs.hubName: %s", err)
		}
	}
}

func withActivityCategories(categories []string) sourceOption {
	// The make slice and for loop is to ensure the string array gets converted to an interface array
	iarray := make([]interface{}, len(categories))
	for i := range categories {
		iarray[i] = categories[i]
	}

	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedSlice(src.Object, iarray, "spec", "categories"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.eventHubID: %s", err)
		}
	}
}

func withSubscriptionID(id string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, id, "spec", "subscriptionID"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.subscriptionID: %s", err)
		}
	}
}

// withServicePrincipal will create the secret and service principal based on the azure environment variables
func withServicePrincipal() sourceOption {
	credsMap := map[string]interface{}{
		"tenantID":     map[string]interface{}{"value": os.Getenv("AZURE_TENANT_ID")},
		"clientID":     map[string]interface{}{"value": os.Getenv("AZURE_CLIENT_ID")},
		"clientSecret": map[string]interface{}{"value": os.Getenv("AZURE_CLIENT_SECRET")},
	}

	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedMap(src.Object, credsMap, "spec", "auth", "servicePrincipal"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.auth.servicePrincipal field: %s", err)
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

// createEventhubID will create the EventHub path used by the k8s azureeventhubssource
func createEventHubNS(subscriptionID, testName string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + testName + "/providers/Microsoft.EventHub/namespaces/" + testName
}
