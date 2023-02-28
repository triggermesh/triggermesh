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

package azurequeuestorage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"

	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-storage-queue-go/azqueue"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	e2eazure "github.com/triggermesh/triggermesh/test/e2e/framework/azure"
	"github.com/triggermesh/triggermesh/test/e2e/framework/bridges"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
)

/*
  This test requires:
  - Azure Service Principal Credentials with the Azure Storage Account role and Azure Queue Storage assigned at the subscription level

  The following environment variables _MUST_ be set:
  - AZURE_SUBSCRIPTION_ID - Common subscription for the test to run against
  - AZURE_TENANT_ID - Azure tenant to create the resources against
  - AZURE_CLIENT_ID - The Azure ServicePrincipal Client ID
  - AZURE_CLIENT_SECRET - The Azure ServicePrincipal Client Secret
  - AZURE_REGION - Define the Azure region to run the test (default uswest2)

  These will be done by the e2e test:
  - Create an Azure Resource Group, Storage Account and a Queue Storage
  - Send a message from the Azure Queue Storage into the TriggerMesh source

*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "AzureQueueStorageSource"
	sourceResource = "azurequeuestoragesource"
)

/*
 Basic flow will resemble:
 * Create a resource group to contain our storage account
 * Create an azure queue storage.
 * Instantiate the AzureQueueStorageSource
 * Send a message to the azure queue storage and look for a response
*/

var _ = Describe("Azure Queue Storage", func() {
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

	BeforeEach(func() {
		ns = f.UniqueName
		gvr := sourceAPIVersion.WithResource(sourceResource + "s")
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source watches a servicebus queue", func() {
		var rg armresources.ResourceGroup
		var queueMessage *azqueue.MessagesURL

		var accountName string
		var accountKey string

		var err error

		SendMessageAndAssertReceivedEvent := func() func() {
			return func() {

				BeforeEach(func() {
					_, err = queueMessage.Enqueue(ctx, "hello world", 0, 0)
					Expect(err).ToNot(HaveOccurred())
				})

				Specify("the source generates an event", func() {
					const receiveTimeout = 250 * time.Second
					const pollInterval = 500 * time.Millisecond

					var receivedEvents []cloudevents.Event

					readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

					Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
					Expect(receivedEvents).To(HaveLen(1))

					e := receivedEvents[0]

					Expect(e.Type()).To(Equal("com.microsoft.azure.queuestorage"))
					Expect(e.Source()).To(Equal(createQueueStorageID(accountName, ns)))

					data := make(map[string]interface{})
					err = json.Unmarshal(e.Data(), &data)
					Expect(err).ToNot(HaveOccurred())

					testID := fmt.Sprintf("%v", data["ID"])
					Expect(data["ID"]).To(Equal(testID))
				})
			}
		}

		BeforeEach(func() {
			// storageaccount name must be alphanumeric characters only and 3-24 characters long
			accountName = strings.Replace(ns, "-", "", -1)
			accountName = strings.Replace(accountName, "e2eazurequeuestoragesource", "tme2etest", -1)

			rg = e2eazure.CreateResourceGroup(ctx, subscriptionID, ns, region)
			DeferCleanup(func() {
				_ = e2eazure.DeleteResourceGroup(ctx, subscriptionID, *rg.Name)
			})

			storageClient := e2eazure.CreateStorageAccountsClient(subscriptionID)

			_ = e2eazure.CreateQueueStorageAccount(ctx, storageClient, accountName, *rg.Name, region)
			Expect(err).ToNot(HaveOccurred())

			keys, err := e2eazure.GetStorageAccountKey(ctx, storageClient, accountName, *rg.Name)
			Expect(err).ToNot(HaveOccurred())

			accountKey = *(keys.Keys)[0].Value

			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
			})
			By("creating a queue storage", func() {
				queueMessage = e2eazure.CreateQueueStorage(ctx, ns, accountName, accountKey)
			})
		})

		Context("the subscription is managed by the source", func() {
			BeforeEach(func() {
				By("creating the AzureQueueStorageSource object", func() {
					src, err := createSource(srcClient, ns, "test-", sink,
						withAccountName(accountName),
						withQueueName(ns),
						withAccountKey(accountKey),
					)
					Expect(err).ToNot(HaveOccurred())

					ducktypes.WaitUntilReady(f.DynamicClient, src)
				})
			})

			When("a message is sent to the queue storage", SendMessageAndAssertReceivedEvent())

		})
	})
})

type sourceOption func(*unstructured.Unstructured)

// createSource creates an AzureQueueStorageSource object initialized with the test parameters
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

func withAccountName(accountName string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, accountName, "spec", "accountName"); err != nil {
			framework.FailfWithOffset(2, "failed to set spec.accountName: %s", err)
		}
	}
}

func withQueueName(queueName string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, queueName, "spec", "queueName"); err != nil {
			framework.FailfWithOffset(2, "failed to set spec.queueName: %s", err)
		}
	}
}

func withAccountKey(accountKey string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, accountKey, "spec", "accountKey", "value"); err != nil {
			framework.FailfWithOffset(2, "failed to set spec.accountKey.value: %s", err)
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

// createQueueStorageID will create the Queue Storage ID
func createQueueStorageID(accountName, name string) string {
	return "https://" + accountName + ".queue.core.windows.net/" + name
}
