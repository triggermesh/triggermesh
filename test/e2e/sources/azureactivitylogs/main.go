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

package azureactivitylogs

import (
	"context"
	"os"
	"time"

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
  - AZURE_REGION - Define the Azure region to run the test (default "westus2")

  These will be done by the e2e test:
  - Create an Azure Resource Group and Event Hubs Namespace
  - Generate an activity inside the Azure subscription

*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "AzureActivityLogsSource"
	sourceResource = "azureactivitylogssources"
)

var activityCategories = []string{"Administrative", "Policy", "Security"}

/*
 Basic flow will resemble:
 * Create a resource group to contain our eventhub
 * Ensure our service principal can read/write from the eventhub
 * Instantiate the AzureActivityLogsSource
 * Create a resource group and watch the event flow in
*/

var _ = Describe("Azure Activity Logs", func() {
	f := framework.New("azureactivitylogssource")

	var ns string

	var srcClient dynamic.ResourceInterface

	var sink *duckv1.Destination
	var subscriptionID string
	var eventHubsNamespaceID string

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := sourceAPIVersion.WithResource(sourceResource)
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source subscribes to activities from an Azure subscription", func() {
		var rgName string
		var eventHubsInstanceName string

		subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")

		region := os.Getenv("AZURE_REGION")
		if region == "" {
			region = "westus2"
		}

		ctx := context.Background()

		BeforeEach(func() {

			By("creating a resource group", func() {
				rg := azure.CreateResourceGroup(ctx, subscriptionID, ns, region)
				rgName = *rg.Name
			})

			By("creating an Event Hubs namespace and instance", func() {
				nsName := f.UniqueName
				_ = azure.CreateEventHubComponents(ctx, subscriptionID, nsName, region, rgName)

				eventHubsNamespaceID = eventHubsNamespaceResourceID(subscriptionID, rgName, nsName)

				// FIXME(antoineco): this assumption might not remain true if CreateEventHubComponents
				// changes in the future. The helper should return the name of the created Event Hubs
				// instance.
				eventHubsInstanceName = nsName
			})

			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
			})

			By("creating an AzureActivityLogsSource object", func() {
				src, err := createSource(srcClient, ns, "test-", sink,
					withServicePrincipal(),
					withSubscriptionID(subscriptionID),
					withActivityCategories(activityCategories),
					withEventHubsNamespaceID(eventHubsNamespaceID),
					withEventHubsInstanceName(eventHubsInstanceName),
				)
				Expect(err).ToNot(HaveOccurred())

				ducktypes.WaitUntilReady(f.DynamicClient, src)
			})
		})

		AfterEach(func() {
			By("deleting the resource group "+rgName, func() {
				_ = azure.DeleteResourceGroup(ctx, subscriptionID, rgName)
			})
		})

		When("an activity is logged inside the subscription", func() {

			// NOTE(antoineco): We create a resource group here to ensure that at least one activity is
			// logged during the runtime of the test. It is worth noting that, in practice, unrelated
			// activities will be generated by simply running other Azure tests in parallel, within the same
			// Azure subscription.

			var testRGName string

			BeforeEach(func() {
				By("creating a sample resource group to produce activity", func() {
					testRG := azure.CreateResourceGroup(ctx, subscriptionID, rgName+"-testrg", region)
					testRGName = *testRG.Name
				})
			})

			AfterEach(func() {
				By("deleting the sample resource group "+testRGName, func() {
					_ = azure.DeleteResourceGroup(ctx, subscriptionID, testRGName)
				})
			})

			Specify("the source generates an event", func() {
				// A latency of ~5 min is expected between the moment an activity is generated, and that
				// activity is sent to the configured destination (Event Hubs).
				const receiveTimeout = 10 * time.Minute
				const pollInterval = 500 * time.Millisecond

				var receivedEvents []cloudevents.Event

				readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

				Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
				e := receivedEvents[0]

				Expect(e.Type()).To(Equal("com.microsoft.azure.monitor.activity-log"))
				Expect(e.Source()).To(Equal("/subscriptions/" + subscriptionID))
			})

		})
	})

	When("a client creates a source object with invalid specs", func() {

		// Those tests do not require a real sink, subscription, or Event Hubs namespace
		BeforeEach(func() {
			subscriptionID = "00000000-0000-0000-0000-000000000000"
			eventHubsNamespaceID = eventHubsNamespaceResourceID(subscriptionID, "test", "test")

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
				invalidSubID := "I'm an invalid subscription"

				_, err := createSource(srcClient, ns, "test-invalid-sub-id", sink,
					withServicePrincipal(),
					withSubscriptionID(invalidSubID),
					withActivityCategories(activityCategories),
					withEventHubsNamespaceID(eventHubsNamespaceID),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.subscriptionID: Invalid value: "))
			})

			By("omitting credentials", func() {
				_, err := createSource(srcClient, ns, "test-nocreds-", sink,
					withSubscriptionID(subscriptionID),
					withActivityCategories(activityCategories),
					withEventHubsNamespaceID(eventHubsNamespaceID),
					withEventHubsInstanceName(ns),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.auth: Required value`))
			})

			By("omitting destination", func() {
				_, err := createSource(srcClient, ns, "test-no-eventhubs-", sink,
					withServicePrincipal(),
					withSubscriptionID(subscriptionID),
					withActivityCategories(activityCategories),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.destination: Required value`))
			})

			By("setting invalid eventhub namespace", func() {
				invalidEventHubsNamespace := "I'm an invalid Event Hubs namespace"

				_, err := createSource(srcClient, ns, "test-invalid-eventhubs-ns-", sink,
					withServicePrincipal(),
					withSubscriptionID(subscriptionID),
					withActivityCategories(activityCategories),
					withEventHubsNamespaceID(invalidEventHubsNamespace),
					withEventHubsInstanceName(ns),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.destination.eventHubs.namespaceID: Invalid value: "`))
			})

			By("setting invalid Event Hubs instance name", func() {
				invalidName := "I'm an invalid Event Hubs instance name"

				_, err := createSource(srcClient, ns, "test-invalid-eventhubs-name-", sink,
					withServicePrincipal(),
					withSubscriptionID(subscriptionID),
					withEventHubsNamespaceID(eventHubsNamespaceID),
					withEventHubsInstanceName(invalidName),
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

func withEventHubsNamespaceID(ns string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, ns, "spec", "destination", "eventHubs", "namespaceID"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.destination.eventHubs.namespaceID: %s", err)
		}
	}
}

func withEventHubsInstanceName(name string) sourceOption {
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
			framework.FailfWithOffset(2, "Failed to set spec.categories: %s", err)
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

// eventHubsNamespaceResourceID returns a fully qualified Azure resource ID for
// an Event Hubs namespace.
func eventHubsNamespaceResourceID(subscriptionID, rgName, nsName string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + rgName +
		"/providers/Microsoft.EventHub/namespaces/" + nsName
}
