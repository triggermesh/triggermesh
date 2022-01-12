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

package azureeventgrid

import (
	"context"
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
  - Microsoft.EventHubs and Microsoft.EventGrid resources to be added to the subscription
  - Microsoft.Eventhubs/write and Microsoft.EventGrid/eventsubscriptions/read and write permissions are required for the
    associated service principal

  The following environment variables _MUST_ be set:
  - AZURE_SUBSCRIPTION_ID - Common subscription for the test to run against
  - AZURE_TENANT_ID - Azure tenant to create the resources against
  - AZURE_CLIENT_ID - The Azure ServicePrincipal Client ID
  - AZURE_CLIENT_SECRET - The Azure ServicePrincipal Client Secret
  - AZURE_REGION - Define the Azure region to run the test (default uswest2)

  These will be done by the e2e test:
  - Create an Azure Resource Group, EventHubs Namespace, and EventHub
  - Register an EventGrid watcher on the resource group
  - Create a storage account and watch for the event

*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "AzureEventGridSource"
	sourceResource = "azureeventgridsource"
)

/*
 Basic flow will resemble:
 * Create a resource group to contain our eventhub
 * Ensure our service principal can read/write from the eventhub
 * Instantiate the AzureEventGridSource
 * Instantiate the Azure Storage Account and verify an event is produced
*/

var _ = Describe("Azure Event Grid", func() {
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

	Context("an Azure Event Grid source is created", func() {
		var err error // stubbed

		BeforeEach(func() {
			rg = azure.CreateResourceGroup(ctx, subscriptionID, ns, region)
			_ = azure.CreateEventHubNamespaceOnly(ctx, subscriptionID, ns, region, *rg.Name)
		})

		When("an event is produced", func() {
			var src *unstructured.Unstructured

			BeforeEach(func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)

				src, err = createSource(srcClient, ns, "test-", sink,
					withServicePrincipal(),
					withEventScope("/subscriptions/"+subscriptionID+"/resourceGroups/"+*rg.Name),
					withEventTypes([]string{"Microsoft.Resources.ResourceWriteSuccess"}),
					withEventHubNamespace(createEventhubID(subscriptionID, ns)),
				)

				Expect(err).ToNot(HaveOccurred())

				ducktypes.WaitUntilReady(f.DynamicClient, src)
				time.Sleep(30 * time.Second) // Will take some extra time to bring up the Azure Eventgrid
			})

			It("should verify an eventgrid event was sent", func() {
				const receiveTimeout = 60 * time.Second
				const pollInterval = 500 * time.Millisecond

				var receivedEvents []cloudevents.Event

				readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

				Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
				Expect(receivedEvents).To(HaveLen(1))

				e := receivedEvents[0]

				Expect(e.Type()).To(Equal("Microsoft.Resources.ResourceWriteSuccess"))
				Expect(strings.ToLower(e.Source())).To(Equal(strings.ToLower("/subscriptions/" + subscriptionID + "/resourceGroups/" + *rg.Name)))
			})
		})

		AfterEach(func() {
			_ = azure.DeleteResourceGroup(ctx, subscriptionID, *rg.Name)
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
			fakeResourceGroupName := "fakegroup"

			By("omitting credentials", func() {
				_, err := createSource(srcClient, ns, "test-empty-credentials", sink,
					withEventScope("/subscriptions/"+subscriptionID+"/resourceGroups/"+fakeResourceGroupName),
					withEventTypes([]string{"Microsoft.Resources.ResourceWriteSuccess"}),
					withEventHubNamespace(createEventhubID(subscriptionID, ns)),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.auth: Required value`))
			})

			By("omitting the scope", func() {
				_, err := createSource(srcClient, ns, "test-empty-scope", sink,
					withServicePrincipal(),
					withEventTypes([]string{"Microsoft.Resources.ResourceWriteSuccess"}),
					withEventHubNamespace(createEventhubID(subscriptionID, ns)),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.scope: Required value`))
			})

			By("omitting the eventhub endpoint", func() {
				_, err := createSource(srcClient, ns, "test-missing-endpoint", sink,
					withServicePrincipal(),
					withEventScope("/subscriptions/"+subscriptionID+"/resourceGroups/"+fakeResourceGroupName),
					withEventTypes([]string{"Microsoft.Resources.ResourceWriteSuccess"}),
				)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.endpoint: Required value`))
			})

			By("setting an invalid eventhub endpoint", func() {
				fakeEventHubNamespace := "I'm a fake eventhub namespace"
				_, err := createSource(srcClient, ns, "test-invalid-eventhub-ns", sink,
					withServicePrincipal(),
					withEventTypes([]string{"Microsoft.Resources.ResourceWriteSuccess"}),
					withEventHubNamespace(fakeEventHubNamespace),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.endpoint.eventHubs.namespaceID: Invalid value: "`))
			})
		})
	})
})

type sourceOption func(*unstructured.Unstructured)

// createSource creates an AzureEventGridSource object initialized with the test parameters
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

func withEventHubNamespace(namespaceID string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, namespaceID, "spec", "endpoint", "eventHubs", "namespaceID"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.endpoint.eventHubs.namespaceID: %s", err)
		}
	}
}

func withEventTypes(eventTypes []string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedStringSlice(src.Object, eventTypes, "spec", "eventTypes"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.eventTypes: %s", err)
		}
	}
}

func withEventScope(eventScope string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, eventScope, "spec", "scope"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.scope: %s", err)
		}
	}
}

// withServicePrincipal will create the service principal component based on the azure environment variables
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

// createEventhubID will create the EventHub path used by the k8s given the subscriptionID and the test unique name
func createEventhubID(subscriptionID, testName string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + testName + "/providers/Microsoft.EventHub/namespaces/" + testName
}
