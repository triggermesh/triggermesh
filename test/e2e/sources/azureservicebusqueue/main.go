/*
Copyright (c) 2021 TriggerMesh Inc.

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

package azureservicebusqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	sv "github.com/Azure/azure-service-bus-go"
	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	e2eazure "github.com/triggermesh/triggermesh/test/e2e/framework/azure"
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
  - Azure Service Principal Credentials with the Azure ServiceBus Data Owner role assigned at the subscription level

  The following environment variables _MUST_ be set:
  - AZURE_SUBSCRIPTION_ID - Common subscription for the test to run against
  - AZURE_TENANT_ID - Azure tenant to create the resources against
  - AZURE_CLIENT_ID - The Azure ServicePrincipal Client ID
  - AZURE_CLIENT_SECRET - The Azure ServicePrincipal Client Secret

  These will be done by the e2e test:
  - Create an Azure Resource Group, ServiceBus Namespace, and a Queue
  - Send an event from the Azure ServiceBus into the TriggerMesh source

*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "AzureServiceBusQueueSource"
	sourceResource = "azureservicebusqueuesource"
)

/*
 Basic flow will resemble:
 * Create a resource group to contain our servicebus
 * Ensure our service principal can read/write from the servicebus
 * Instantiate the AzureServiceBusQueueSource
 * Send an event to the AzureServiceBusQueueSource and look for a response
*/

var _ = Describe("Azure ServiceBusQueue", func() {
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
	var queue *sv.QueueEntity
	var queueSender *sv.Queue

	BeforeEach(func() {
		ns = f.UniqueName
		gvr := sourceAPIVersion.WithResource(sourceResource + "s")
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source watches a servicebus queue", func() {
		var err error // stubbed

		When("an event flows", func() {
			BeforeEach(func() {
				rg = e2eazure.CreateResourceGroup(ctx, subscriptionID, ns, region)
				nsClient := e2eazure.CreateServiceBusNamespaceClient(ctx, subscriptionID, ns)
				err := e2eazure.CreateServiceBusNamespace(ctx, *nsClient, *rg.Name, ns, region)
				Expect(err).ToNot(HaveOccurred())
				queue, queueSender = createQueue(ctx, region, ns, nsClient)
			})

			It("should create an azure servicebus queue subscription", func() {
				By("creating an event sink", func() {
					sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
				})
				var src *unstructured.Unstructured
				By("creating the azureservicebussource", func() {
					src, err = createSource(srcClient, ns, "test-", sink,
						withServicePrincipal(),
						withSubscriptionID(subscriptionID),
						withQueueID(createQueueID(subscriptionID, ns, queue.Name)),
					)
					Expect(err).ToNot(HaveOccurred())

					ducktypes.WaitUntilReady(f.DynamicClient, src)
				})

				By("sending an event", func() {
					err = queueSender.Send(ctx, sv.NewMessageFromString("hello world"))
					Expect(err).ToNot(HaveOccurred())
				})

				By("verifying the event was sent", func() {
					const receiveTimeout = 25 * time.Second // It needs some more time than our default 15s.
					const pollInterval = 500 * time.Millisecond

					var receivedEvents []cloudevents.Event

					readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

					Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
					Expect(receivedEvents).To(HaveLen(1))

					e := receivedEvents[0]

					Expect(e.Type()).To(Equal("com.microsoft.azure.servicebus.message"))

					data := make(map[string]interface{})
					err = json.Unmarshal(e.Data(), &data)
					testID := fmt.Sprintf("%v", data["ID"])
					Expect(data["ID"]).To(Equal(testID))
				})
			})

			AfterEach(func() {
				_ = e2eazure.DeleteResourceGroup(ctx, subscriptionID, *rg.Name)
			})
		})
	})

	When("a client creates a source object with invalid specs", func() {
		var fakeQueueID string

		// Those tests do not require a real sink
		BeforeEach(func() {
			sink = &duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: "fake/v1",
					Kind:       "Fake",
					Name:       "fake",
				},
			}

			fakeQueueID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ServiceBus/namespaces/%s/queues/%s", subscriptionID, ns, ns, "fakeQueue")
		})

		Specify("the API server rejects the creation of that object", func() {
			By("omitting credentials", func() {
				_, err := createSource(srcClient, ns, "test-empty-credentials", sink,
					withSubscriptionID(subscriptionID),
					withQueueID(fakeQueueID),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.auth: Required value`))
			})

			By("setting an invalid queue name", func() {
				_, err := createSource(srcClient, ns, "test-invalid-queueName", sink,
					withSubscriptionID(subscriptionID),
					withQueueID("fakename"),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`spec.queueID: Invalid value: "`))
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

func withQueueID(id string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, id, "spec", "queueID"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.queueID: %s", err)
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

// createQueue will create a servicebus queue and queueSender using the given name
func createQueue(ctx context.Context, region string, name string, nsCli *servicebus.NamespacesClient) (*sv.QueueEntity, *sv.Queue) {
	keys, err := nsCli.ListKeys(ctx, name, name, "RootManageSharedAccessKey")
	if err != nil {
		framework.FailfWithOffset(2, "Unable to obtain the connection string: %s", err)
		return nil, nil
	}

	// Take the namespace connection string, and add the specific servicehub
	connectionString := *keys.PrimaryConnectionString + ";EntityPath=" + name
	svNs := sv.NamespaceWithConnectionString(connectionString)
	if svNs == nil {
		framework.FailfWithOffset(2, "Unable to configure the namespace client: %s", err)
		return nil, nil
	}

	ns, err := sv.NewNamespace(svNs)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to create the namespace client: %s", err)
		return nil, nil
	}

	tm := ns.NewQueueManager()

	queue, err := tm.Put(ctx, name)
	if err != nil {
		framework.FailfWithOffset(2, "Error creating queue: %s", err)
		return nil, nil
	}

	queueSender, err := ns.NewQueue(queue.Name)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to create the queue sender: %s", err)
		return nil, nil
	}

	return queue, queueSender
}

// createQueueID will create the queueID path used by the k8s azureservicebusqueuesource
func createQueueID(subscriptionID, name, queueName string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + name + "/providers/Microsoft.ServiceBus/namespaces/" + name + "/queues/" + queueName
}
