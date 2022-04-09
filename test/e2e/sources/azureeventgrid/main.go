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

package azureeventgrid

import (
	"bytes"
	"context"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck
	"github.com/onsi/gomega/types"

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
  - Microsoft.EventHubs and Microsoft.EventGrid resources to be added to the subscription
  - Microsoft.Eventhubs/write and Microsoft.EventGrid/eventsubscriptions/read and write permissions are required for the
    associated service principal

  The following environment variables _MUST_ be set:
  - AZURE_SUBSCRIPTION_ID - Common subscription for the test to run against
  - AZURE_TENANT_ID - Azure tenant to create the resources against
  - AZURE_CLIENT_ID - The Azure ServicePrincipal Client ID
  - AZURE_CLIENT_SECRET - The Azure ServicePrincipal Client Secret
  - AZURE_REGION - Define the Azure region to run the test (default "westus2")

  These will be done by the e2e test:
  - Create an Azure Resource Group, Event Hubs Namespace, and Event Hubs instance
  - Create an AzureEventGridSource instance that subscribes to events from the resource group
  - Create an Azure Storage Account to generate an event inside the resource group

*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "AzureEventGridSource"
	sourceResource = "azureeventgridsources"
)

/*
 Basic flow will resemble:
 * Create a resource group to contain our eventhub
 * Ensure our service principal can read/write from the eventhub
 * Instantiate the AzureEventGridSource
 * Instantiate the Azure Storage Account and verify an event is produced
*/

var _ = Describe("Azure Event Grid source", func() {
	f := framework.New("azureeventgridsource")

	var ns string

	var srcClient dynamic.ResourceInterface

	var sink *duckv1.Destination
	var eventScope string
	var eventHubsNamespaceID string

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := sourceAPIVersion.WithResource(sourceResource)
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source subscribes to events from a resource group", func() {
		var rgName string

		subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

		region := os.Getenv("AZURE_REGION")
		if region == "" {
			region = "westus2"
		}

		ctx := context.Background()

		BeforeEach(func() {

			By("creating a resource group", func() {
				rg := azure.CreateResourceGroup(ctx, subscriptionID, ns, region)
				rgName = *rg.Name

				eventScope = "/subscriptions/" + subscriptionID + "/resourceGroups/" + *rg.Name

				DeferCleanup(func() {
					By("deleting the resource group "+rgName, func() {
						_ = azure.DeleteResourceGroup(ctx, subscriptionID, rgName)
					})
				})
			})

			By("creating an Event Hubs namespace", func() {
				nsName := f.UniqueName
				_ = azure.CreateEventHubNamespaceOnly(ctx, subscriptionID, nsName, region, rgName)

				eventHubsNamespaceID = eventHubsNamespaceResourceID(subscriptionID, rgName, nsName)
			})

			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
			})

			By("creating an AzureEventGridSource object", func() {
				src, err := createSource(srcClient, ns, "test-", sink,
					withServicePrincipal(),
					withEventScope(eventScope),
					withEventTypes([]string{"Microsoft.Resources.ResourceWriteSuccess"}),
					withEventHubsNamespaceID(eventHubsNamespaceID),
				)
				Expect(err).ToNot(HaveOccurred())

				ducktypes.WaitUntilReady(f.DynamicClient, src)
			})
		})

		When("an event is generated inside the resource group", func() {
			var sampleStorageAccountID string

			BeforeEach(func() {
				By("creating a storage account to generate an event", func() {
					storCli := azure.CreateStorageAccountsClient(subscriptionID)
					storageAccount := azure.CreateBlobStorageAccount(ctx, storCli, rgName, region)

					sampleStorageAccountID = *storageAccount.ID
				})
			})

			Specify("the source generates an event", func() {
				// There can be a significant delay (1-10 min) between the moment an Azure resource is
				// created and Event Grid emits the corresponding 'ResourceWriteSuccess' event.
				const receiveTimeout = 10 * time.Minute
				const pollInterval = 500 * time.Millisecond

				var receivedEvents []cloudevents.Event

				// NOTE(antoineco): Although we only create one resource inside our resource group, it
				// is very likely that we receive unrelated events generated by Azure itself while
				// setting up the Event Grid subscription.
				// For this reason, we perform some extra filtering below to ensure that subsequent
				// assertions are performed on the expected event only.
				readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents,
					cloudEventDataContains([]byte(`"operationName":"Microsoft.Storage/storageAccounts/write"`)),
				)
				Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
				Expect(receivedEvents).To(HaveLen(1))

				e := receivedEvents[0]

				Expect(e.Type()).To(Equal("Microsoft.Resources.ResourceWriteSuccess"))
				Expect(e.Source()).To(equalResourceID(eventScope))
				Expect(e.Subject()).To(equalResourceID(sampleStorageAccountID))
			})

		})
	})

	When("a client creates a source object with invalid specs", func() {

		// Those tests do not require a real sink, scope, or Event Hubs namespace
		BeforeEach(func() {
			const subscriptionID = "00000000-0000-0000-0000-000000000000"
			eventScope = "/subscriptions/" + subscriptionID
			eventHubsNamespaceID = eventHubsNamespaceResourceID(subscriptionID, "testRg", "testNs")

			sink = &duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: "fake/v1",
					Kind:       "Fake",
					Name:       "fake",
				},
			}
		})

		Specify("the API server rejects the creation of that object", func() {

			By("setting an invalid scope", func() {
				invalidScopeResourceID := "I'm an invalid resource ID"

				_, err := createSource(srcClient, ns, "test-invalid-scope", sink,
					withServicePrincipal(),
					withEventScope(invalidScopeResourceID),
					withEventHubsNamespaceID(eventHubsNamespaceID),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.scope: Invalid value: "))
			})

			By("omitting credentials", func() {
				_, err := createSource(srcClient, ns, "test-nocreds-", sink,
					withEventScope(eventScope),
					withEventHubsNamespaceID(eventHubsNamespaceID),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.auth: Required value`))
			})

			By("omitting the Event Hubs endpoint", func() {
				_, err := createSource(srcClient, ns, "test-no-eventhubs-", sink,
					withServicePrincipal(),
					withEventScope(eventScope),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.endpoint: Required value`))
			})

			By("setting an invalid Event Hubs namespace", func() {
				invalidEventHubsNamespace := "I'm an invalid Event Hubs namespace"

				_, err := createSource(srcClient, ns, "test-invalid-eventhubs-ns-", sink,
					withServicePrincipal(),
					withEventScope(eventScope),
					withEventHubsNamespaceID(invalidEventHubsNamespace),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.endpoint.eventHubs.namespaceID: Invalid value: "`))
			})

			By("setting an invalid Event Hubs instance name", func() {
				invalidName := "I'm an invalid Event Hubs instance name"

				_, err := createSource(srcClient, ns, "test-invalid-eventhubs-name-", sink,
					withServicePrincipal(),
					withEventScope(eventScope),
					withEventHubsNamespaceID(eventHubsNamespaceID),
					withEventHubsInstanceName(invalidName),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.endpoint.eventHubs.hubName: Invalid value: "`))
			})
		})
	})
})

type sourceOption func(*unstructured.Unstructured)

// createSource creates an AzureEventGridSource object initialized with the given options.
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
		if err := unstructured.SetNestedField(src.Object, ns, "spec", "endpoint", "eventHubs", "namespaceID"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.endpoint.eventHubs.namespaceID: %s", err)
		}
	}
}

func withEventHubsInstanceName(name string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, name, "spec", "endpoint", "eventHubs", "hubName"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.endpoint.eventHubs.hubName: %s", err)
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
	receivedEvents *[]cloudevents.Event, filterFn cloudEventPredicate) func() []cloudevents.Event {

	return func() []cloudevents.Event {
		ev := bridges.ReceivedEventDisplayEvents(
			apps.GetLogs(c, namespace, eventDisplayName),
		)

		if filterFn != nil {
			filteredEv := ev[:0]

			for _, e := range ev {
				if filterFn(e) {
					filteredEv = append(filteredEv, e)
				}
			}

			ev = filteredEv
		}

		*receivedEvents = ev

		return ev
	}
}

// cloudEventPredicate is a predicate function that can be used as a filter to
// verify different aspects of a CloudEvent.
type cloudEventPredicate func(cloudevents.Event) bool

// cloudEventDataContains returns a predicate function which asserts that a
// CloudEvent's data contains the given bytes.
func cloudEventDataContains(subslice []byte) cloudEventPredicate {
	return func(e cloudevents.Event) bool {
		return bytes.Contains(e.Data(), subslice)
	}
}

// eventHubsNamespaceResourceID returns a fully qualified Azure resource ID for
// an Event Hubs namespace.
func eventHubsNamespaceResourceID(subscriptionID, rgName, nsName string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + rgName +
		"/providers/Microsoft.EventHub/namespaces/" + nsName
}

// equalResourceID returns a Gomega matcher which asserts the equality of two
// Azure resource IDs.
// Unlike gomega.Equal, this function accounts for Event Grid generating
// CloudEvent attributes with an inconsistent casing of the "resourceGroups"
// segment of resource IDs.
func equalResourceID(expectID string) types.GomegaMatcher {
	return SatisfyAny(
		Equal(expectID),
		Equal(strings.Replace(expectID, "/resourceGroups/", "/resourcegroups/", 1)),
	)
}
