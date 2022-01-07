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

package azureblobstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/go-autorest/autorest/to"

	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

	"github.com/triggermesh/triggermesh/test/e2e/framework/azure"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	"github.com/triggermesh/triggermesh/test/e2e/framework/bridges"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	duckv1 "knative.dev/pkg/apis/duck/v1"
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
  - Setup Azure Blob Storage and configure the events to stream to the Azure EventHub

*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind          = "AzureBlobStorageSource"
	sourceResource      = "azureblobstoragesource"
	azureBlobStorageURL = ".blob.core.windows.net/"
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

var _ = Describe("Azure Blob Storage", func() {
	ctx := context.Background()
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	region := os.Getenv("AZURE_REGION")

	if region == "" {
		region = "westus2"
	}

	f := framework.New(sourceResource)

	var ns string
	var saName string
	var srcClient dynamic.ResourceInterface
	var sink *duckv1.Destination

	var rg armresources.ResourceGroup

	var sa armstorage.StorageAccount
	var container armstorage.BlobContainer

	BeforeEach(func() {
		ns = f.UniqueName
		gvr := sourceAPIVersion.WithResource(sourceResource + "s")
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("an Azure Blob is created and deleted ", func() {
		var err error // stubbed

		When("a blob is created and deleted", func() {
			BeforeEach(func() {
				rg = azure.CreateResourceGroup(ctx, subscriptionID, ns, region)
				_ = azure.CreateEventHubNamespaceOnly(ctx, subscriptionID, ns, region, *rg.Name)
			})

			var src *unstructured.Unstructured

			BeforeEach(func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)

				// storageaccount name must be alphanumeric characters only and 3-24 characters long
				saName = strings.Replace(ns, "-", "", -1)
				saName = strings.Replace(saName, "e2eazureblobstoragesource", "tme2etest", -1)
				sa = createStorageAccount(ctx, subscriptionID, *rg.Name, saName, region)

				container = createBlobContainer(ctx, *rg.Name, sa, subscriptionID, ns)

				src, err = createSource(srcClient, ns, "test-", sink,
					withServicePrincipal(),
					withEventTypes([]string{"Microsoft.Storage.BlobCreated", "Microsoft.Storage.BlobDeleted"}),
					withEventHubNamespace(createEventhubID(subscriptionID, ns)),
					withStorageAccountID(createStorageAccountID(subscriptionID, ns, saName)),
				)

				Expect(err).ToNot(HaveOccurred())

				ducktypes.WaitUntilReady(f.DynamicClient, src)
				time.Sleep(30 * time.Second) // Will take some extra time to bring up the Azure Eventgrid
			})

			It("should verify an Azure storage event is sent", func() {
				By("uploading a blob", func() {
					uploadBlob(ctx, container, sa, ns, generatePayload(4096))
					time.Sleep(30 * time.Second) // wait for the blob to be created

					const receiveTimeout = 60 * time.Second // it takes events a little longer to flow in from azure
					const pollInterval = 500 * time.Millisecond

					var receivedEvents []cloudevents.Event

					readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

					Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
					Expect(receivedEvents).To(HaveLen(1))

					e := receivedEvents[0]

					Expect(e.Type()).To(Equal("Microsoft.Storage.BlobCreated"))
					Expect(e.Source()).To(Equal(createStorageAccountID(subscriptionID, ns, saName)))

					// Verify the put request
					var data map[string]interface{}
					err = json.Unmarshal(e.Data(), &data)
					Expect(err).ToNot(HaveOccurred())

					Expect(data["api"]).To(Equal("PutBlob"))
					Expect(data["url"]).To(Equal("https://" + *sa.Name + azureBlobStorageURL + *container.Name + "/" + ns))
				})

				By("deleting a blob", func() {
					deleteBlob(ctx, container, sa, ns)
					time.Sleep(60 * time.Second) // wait for the blob to be deleted

					const receiveTimeout = 60 * time.Second // it takes events a little longer to flow in from azure
					const pollInterval = 500 * time.Millisecond

					var receivedEvents []cloudevents.Event

					readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

					Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
					Expect(receivedEvents).To(HaveLen(2))

					e := receivedEvents[1]

					Expect(e.Type()).To(Equal("Microsoft.Storage.BlobDeleted"))
					Expect(e.Source()).To(Equal(createStorageAccountID(subscriptionID, ns, saName)))

					// Verify the put request
					var data map[string]interface{}
					err = json.Unmarshal(e.Data(), &data)
					Expect(err).ToNot(HaveOccurred())

					Expect(data["api"]).To(Equal("DeleteBlob"))
					Expect(data["url"]).To(Equal("https://" + *sa.Name + azureBlobStorageURL + *container.Name + "/" + ns))
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

			saName = strings.Replace(ns, "-", "", -1)
			saName = strings.Replace(saName, "e2eazureblobstoragesource", "fakesaname", -1)
		})

		Specify("the API server rejects the creation of that object", func() {

			By("omitting credentials", func() {
				_, err := createSource(srcClient, ns, "test-empty-credentials", sink,
					withEventHubNamespace(createEventhubID(subscriptionID, ns)),
					withStorageAccountID(createStorageAccountID(subscriptionID, ns, saName)),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.auth: Required value`))
			})

			By("omitting storageAccountID", func() {
				_, err := createSource(srcClient, ns, "test-empty-storage-id", sink,
					withServicePrincipal(),
					withEventTypes([]string{"Microsoft.Storage.BlobCreated", "Microsoft.Storage.BlobDeleted"}),
					withEventHubNamespace(createEventhubID(subscriptionID, ns)),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.storageAccountID: Required value`))
			})

			By("setting invalid storageAccountID", func() {
				fakeStorageAccountID := "I'm a fake storage account"
				_, err := createSource(srcClient, ns, "test-invalid-storageAccountID", sink,
					withServicePrincipal(),
					withEventTypes([]string{"Microsoft.Storage.BlobCreated", "Microsoft.Storage.BlobDeleted"}),
					withEventHubNamespace(createEventhubID(subscriptionID, ns)),
					withStorageAccountID(fakeStorageAccountID),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.storageAccountID: Invalid value: "`))
			})

			By("omitting the eventhub endpoint", func() {
				_, err := createSource(srcClient, ns, "test-missing-endpoint", sink,
					withServicePrincipal(),
					withEventTypes([]string{"Microsoft.Storage.BlobCreated", "Microsoft.Storage.BlobDeleted"}),
					withStorageAccountID(createStorageAccountID(subscriptionID, ns, saName)),
				)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.endpoint: Required value`))
			})

			By("setting invalid eventhub endpoint", func() {
				fakeEventHubNamespace := "I'm a fake eventhub namespace"
				_, err := createSource(srcClient, ns, "test-invalid-eventhub-ns", sink,
					withServicePrincipal(),
					withEventTypes([]string{"Microsoft.Storage.BlobCreated", "Microsoft.Storage.BlobDeleted"}),
					withEventHubNamespace(fakeEventHubNamespace),
					withStorageAccountID(createStorageAccountID(subscriptionID, ns, saName)),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.endpoint.eventHubs.namespaceID: Invalid value: "`))
			})

			By("setting invalid eventTypes", func() {
				fakeEventTypes := []string{"Microsoft.EventGrid.All"}
				_, err := createSource(srcClient, ns, "test-invalid-eventtype", sink,
					withServicePrincipal(),
					withEventTypes(fakeEventTypes),
					withEventHubNamespace(createEventhubID(subscriptionID, ns)),
					withStorageAccountID(createStorageAccountID(subscriptionID, ns, saName)),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(`spec.eventTypes: Unsupported value: "`))
			})
		})
	})
})

type sourceOption func(*unstructured.Unstructured)

// createSource creates an AzureBlobStorageSource object initialized with the test parameters
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

func withStorageAccountID(id string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, id, "spec", "storageAccountID"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.storageAccountID: %s", err)
		}
	}
}

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

// createStorageAccountID will create the StorageAccountID path used by the k8s
func createStorageAccountID(subscriptionID, rgName, saName string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + rgName + "/providers/Microsoft.Storage/storageAccounts/" + saName
}

// createStorageBlobAccount creates a new storage account and blob container to
// user for the test.
func createStorageAccount(ctx context.Context, subscriptionID, resourceGroup, name, region string) armstorage.StorageAccount {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(3, "unable to authenticate: %s", err)
	}

	saClient := armstorage.NewStorageAccountsClient(subscriptionID, cred, nil)

	resp, err := saClient.BeginCreate(ctx, resourceGroup, name, armstorage.StorageAccountCreateParameters{
		Kind:     armstorage.KindBlobStorage.ToPtr(),
		Location: &region,
		SKU: &armstorage.SKU{
			Name: armstorage.SKUNameStandardRAGRS.ToPtr(),
			Tier: armstorage.SKUTierStandard.ToPtr(),
		},
		Identity: &armstorage.Identity{
			Type: armstorage.IdentityTypeNone.ToPtr(),
		},
		Properties: &armstorage.StorageAccountPropertiesCreateParameters{
			AccessTier:            armstorage.AccessTierHot.ToPtr(),
			AllowBlobPublicAccess: to.BoolPtr(true),
		},
	}, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to create storage account: %s", err)
	}

	finalResp, err := resp.PollUntilDone(ctx, time.Second*30)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to create storage account: %s", err)
	}

	return finalResp.StorageAccount
}

func createBlobContainer(ctx context.Context, rg string, sa armstorage.StorageAccount, subscriptionID, name string) armstorage.BlobContainer {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to authenticate: %s", err)
	}

	client := armstorage.NewBlobContainersClient(subscriptionID, cred, nil)

	resp, err := client.Create(ctx, rg, *sa.Name, name, armstorage.BlobContainer{}, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to create blob container: %s", err)
	}

	return resp.BlobContainer
}

func uploadBlob(ctx context.Context, container armstorage.BlobContainer, sa armstorage.StorageAccount, name string, data string) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to authenticate: %s", err)
	}

	url := "https://" + *sa.Name + azureBlobStorageURL + *container.Name
	containerClient, err := azblob.NewContainerClient(url, cred, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to obtain blob client: %s", err)
	}

	blobClient := containerClient.NewBlockBlobClient(name)
	rs := ReadSeekCloser(strings.NewReader(data))

	_, err = blobClient.Upload(ctx, rs, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to upload payload: %s", err)
	}
}

func deleteBlob(ctx context.Context, container armstorage.BlobContainer, sa armstorage.StorageAccount, name string) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to authenticate: %s", err)
	}

	url := "https://" + *sa.Name + azureBlobStorageURL + *container.Name
	containerClient, err := azblob.NewContainerClient(url, cred, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to obtain blob client: %s", err)
	}

	blobClient := containerClient.NewBlockBlobClient(name)
	_, err = blobClient.Delete(ctx, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to delete blob: %s", err)
	}
}

func generatePayload(size int) string {
	var sb strings.Builder

	for i := 0; i < size; i++ {
		s := fmt.Sprintf("%d", rand.Int31())
		sb.WriteString(s)
	}

	return sb.String()
}

// ReadSeekCloser implements a closer with Seek, Read, and Close
func ReadSeekCloser(r *strings.Reader) readSeekCloser {
	return readSeekCloser{r}
}

type readSeekCloser struct {
	*strings.Reader
}

func (readSeekCloser) Close() error { return nil }
