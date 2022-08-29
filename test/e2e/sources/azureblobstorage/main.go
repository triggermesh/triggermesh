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

package azureblobstorage

import (
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
  - Create an Azure Storage Account
  - Create an AzureBlobStorageSource instance that subscribes to events from the storage account

*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "AzureBlobStorageSource"
	sourceResource = "azureblobstoragesources"
)

/*
 Basic flow will resemble:
 * Create a resource group to contain our eventhub
 * Ensure our service principal can read/write from the eventhub
 * Instantiate the AzureBlobStorageSource
 * Instantiate the Azure Storage Account and create a container for our blob
 * Create a new file, upload it to the blob, verify the event
 * Delete the blob and verify the event
*/

var _ = Describe("Azure Blob Storage source", func() {
	f := framework.New("azureblobstoragesource")

	var ns string

	var srcClient dynamic.ResourceInterface

	var sink *duckv1.Destination
	var storageAccountID string
	var eventHubsNamespaceID string

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := sourceAPIVersion.WithResource(sourceResource)
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source subscribes to events from a blob container", func() {
		var rgName string
		var storageAccountName string
		var blobContainerName string

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

				DeferCleanup(func() {
					By("deleting the resource group "+rgName, func() {
						_ = azure.DeleteResourceGroup(ctx, subscriptionID, rgName)
					})
				})
			})

			By("creating a storage account and Blob container", func() {
				storCli := azure.CreateStorageAccountsClient(subscriptionID)
				storageAccount := azure.CreateBlobStorageAccount(ctx, storCli, rgName, region)
				storageAccountID = *storageAccount.ID
				storageAccountName = *storageAccount.Name

				container := azure.CreateBlobContainer(ctx, rgName, *storageAccount.Name, subscriptionID, "e2e")
				blobContainerName = *container.Name
			})

			By("creating an Event Hubs namespace", func() {
				nsName := f.UniqueName
				_ = azure.CreateEventHubNamespaceOnly(ctx, subscriptionID, nsName, region, rgName)

				eventHubsNamespaceID = eventHubsNamespaceResourceID(subscriptionID, rgName, nsName)
			})

			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
			})

			By("creating an AzureBlobStorageSource object", func() {
				src, err := createSource(srcClient, ns, "test-", sink,
					withServicePrincipal(),
					withStorageAccount(storageAccountID),
					withEventTypes([]string{"Microsoft.Storage.BlobCreated", "Microsoft.Storage.BlobDeleted"}),
					withEventHubsNamespaceID(eventHubsNamespaceID),
				)
				Expect(err).ToNot(HaveOccurred())

				ducktypes.WaitUntilReady(f.DynamicClient, src)

				// FIXME(antoineco): Azure needs some extra time for setting up Event Grid's system
				// topic upon creation of the Event Grid subscription by our reconciler (Blob Storage
				// events are routed over Event Grid). The source shouldn't report Ready before this
				// system topic is available, because events occuring prior to that are dropped.
				// Ref. https://github.com/triggermesh/triggermesh/issues/446
				time.Sleep(1 * time.Minute)
			})
		})

		When("blobs are created and deleted", func() {

			// This test is structured as
			//   "Specify: the source generates ..., By: creating/deleting a blob ..."
			// instead of
			//   "When: a blob is created/deleted ..., Specify: the source generates ..."
			// to avoid creating a separate set of Azure resources for each spec, which would significantly
			// increase the duration of the test with no real benefit.
			Specify("the source generates events", func() {

				By("creating a blob", func() {
					azure.UploadBlob(ctx, blobContainerName, storageAccountName, "hello.txt", "Hello, World!")
				})

				By("asserting that a BlobCreated event is received", func() {
					const receiveTimeout = 15 * time.Second
					const pollInterval = 500 * time.Millisecond

					var receivedEvents []cloudevents.Event

					readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

					Eventually(readReceivedEvents, receiveTimeout, pollInterval).Should(HaveLen(1))

					e := receivedEvents[0]

					Expect(e.Type()).To(Equal("Microsoft.Storage.BlobCreated"))
					Expect(e.Source()).To(equalResourceID(storageAccountID))
					Expect(e.Subject()).To(Equal("/blobServices/default/containers/" + blobContainerName + "/blobs/hello.txt"))
				})

				By("deleting the blob", func() {
					azure.DeleteBlob(ctx, blobContainerName, storageAccountName, "hello.txt")
				})

				By("asserting that a BlobDeleted event is received", func() {
					const receiveTimeout = 15 * time.Second
					const pollInterval = 500 * time.Millisecond

					var receivedEvents []cloudevents.Event

					readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

					Eventually(readReceivedEvents, receiveTimeout, pollInterval).Should(HaveLen(2))

					e := receivedEvents[1]

					Expect(e.Type()).To(Equal("Microsoft.Storage.BlobDeleted"))
					Expect(e.Source()).To(equalResourceID(storageAccountID))
					Expect(e.Subject()).To(Equal("/blobServices/default/containers/" + blobContainerName + "/blobs/hello.txt"))
				})
			})
		})
	})

	When("a client creates a source object with invalid specs", func() {

		// Those tests do not require a real sink, storage account, or Event Hubs namespace
		BeforeEach(func() {
			const subscriptionID = "00000000-0000-0000-0000-000000000000"
			storageAccountID = storageAccountResourceID(subscriptionID, "testRg", "teststoracc")
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

			By("setting an invalid storage account", func() {
				invalidStorAccResourceID := "I'm an invalid resource ID"

				_, err := createSource(srcClient, ns, "test-invalid-storacc", sink,
					withServicePrincipal(),
					withStorageAccount(invalidStorAccResourceID),
					withEventHubsNamespaceID(eventHubsNamespaceID),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.storageAccountID: Invalid value: "))
			})

			By("setting unsupported event types", func() {
				_, err := createSource(srcClient, ns, "test-invalid-eventtypes", sink,
					withServicePrincipal(),
					withStorageAccount(storageAccountID),
					withEventTypes([]string{"Microsoft.NotStorage.FakeEvent"}),
					withEventHubsNamespaceID(eventHubsNamespaceID),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.eventTypes[0]: Unsupported value: "Microsoft.NotStorage.FakeEvent"`))
			})

			By("omitting credentials", func() {
				_, err := createSource(srcClient, ns, "test-nocreds-", sink,
					withStorageAccount(storageAccountID),
					withEventHubsNamespaceID(eventHubsNamespaceID),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.auth: Required value`))
			})

			By("omitting the Event Hubs endpoint", func() {
				_, err := createSource(srcClient, ns, "test-no-eventhubs-", sink,
					withServicePrincipal(),
					withStorageAccount(storageAccountID),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.endpoint: Required value`))
			})

			By("setting an invalid Event Hubs namespace", func() {
				invalidEventHubsNamespace := "I'm an invalid Event Hubs namespace"

				_, err := createSource(srcClient, ns, "test-invalid-eventhubs-ns-", sink,
					withServicePrincipal(),
					withStorageAccount(storageAccountID),
					withEventHubsNamespaceID(invalidEventHubsNamespace),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.endpoint.eventHubs.namespaceID: Invalid value: "`))
			})

			By("setting an invalid Event Hubs instance name", func() {
				invalidName := "I'm an invalid Event Hubs instance name"

				_, err := createSource(srcClient, ns, "test-invalid-eventhubs-name-", sink,
					withServicePrincipal(),
					withStorageAccount(storageAccountID),
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

// createSource creates an AzureBlobStorageSource object initialized with the given options.
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

func withStorageAccount(storAccID string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, storAccID, "spec", "storageAccountID"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.storageAccountID: %s", err)
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

// storageAccountResourceID returns a fully qualified Azure resource ID for a
// Storage Account.
func storageAccountResourceID(subscriptionID, rgName, saName string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + rgName +
		"/providers/Microsoft.Storage/storageAccounts/" + saName
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
