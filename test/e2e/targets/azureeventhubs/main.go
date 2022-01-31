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

package azureeventhubs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/Azure/azure-event-hubs-go/v3/persist"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	eventhubs "github.com/Azure/azure-event-hubs-go/v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	"github.com/triggermesh/triggermesh/test/e2e/framework/azure"
	e2ece "github.com/triggermesh/triggermesh/test/e2e/framework/cloudevents"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
)

var targetAPIVersion = schema.GroupVersion{
	Group:   "targets.triggermesh.io",
	Version: "v1alpha1",
}

const (
	targetKind = "AzureEventHubsTarget"
)

/*
  This test requires:
  - Azure Service Principal Credentials with the Azure Event Hubs Data Owner role assigned at the subscription level
  The following environment variables _MUST_ be set:
  - AZURE_SUBSCRIPTION_ID - Common subscription for the test to run against
  - AZURE_TENANT_ID - Azure tenant to create the resources against
  - AZURE_CLIENT_ID - The Azure ServicePrincipal Client ID
  - AZURE_CLIENT_SECRET - The Azure ServicePrincipal Client Secret
  - AZURE_REGION - Define the Azure region to run the test (default westus2)

  These will be done by the e2e test:
  - Create an Azure Resource Group, Event Hubs Namespace, and Event Hub
  - Send an event to the Azure Event Hub via the Azure Event Hub Target and verify the event was sent
*/

var _ = Describe("Azure Event Hubs target", func() {
	ctx := context.Background()
	f := framework.New("azureeventhubstarget")

	var ns string
	var tgtClient dynamic.ResourceInterface

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := targetAPIVersion.WithResource("azureeventhubstargets")
		tgtClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a target is deployed", func() {
		var rg string
		var hub *eventhubs.Hub

		subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
		region := os.Getenv("AZURE_REGION")

		if region == "" {
			region = "westus2"
		}

		BeforeEach(func() {
			By("creating an Azure resource group", func() {
				r := azure.CreateResourceGroup(ctx, subscriptionID, ns, region)
				rg = *r.Name
				DeferCleanup(func() {
					By("deleting the Azure resource group", func() {
						_ = azure.DeleteResourceGroup(ctx, subscriptionID, rg)
					})
				})
			})

			By("creating an Azure Event Hub", func() {
				hub = azure.CreateEventHubComponents(ctx, subscriptionID, ns, region, rg)
			})
		})

		When("the spec contains default settings", func() {
			var tgtURL *url.URL

			BeforeEach(func() {
				By("creating an AzureEventHubsTarget object", func() {
					tgt, err := createTarget(tgtClient, ns, "test-",
						withServicePrincipal(),
						withEventHubID(subscriptionID, rg, ns, ns))

					Expect(err).ToNot(HaveOccurred())

					tgt = ducktypes.WaitUntilReady(f.DynamicClient, tgt)
					tgtURL = ducktypes.Address(tgt)
					Expect(tgtURL).ToNot(BeNil())
				})
			})

			It("receives an event on the Event Hub", func() {
				var event *cloudevents.Event
				var partitionIDs []string
				var eventHandler eventhubs.Handler
				var evCtx context.Context // Used to set a timeout for reading events
				var payload []byte

				eventReceivedChannel := make(chan bool)

				By("retrieving Event Hub partition details", func() {
					info, err := hub.GetRuntimeInformation(ctx)
					Expect(err).NotTo(HaveOccurred())

					partitionIDs = info.PartitionIDs
					Expect(len(partitionIDs)).To(BeNumerically(">", 0))
				})

				By("setting up a handler to verify the received event", func() {
					eventHandler = func(ctx context.Context, ev *eventhubs.Event) error {
						payload = ev.Data

						// Pass the bool to the channel to terminate the receiver
						eventReceivedChannel <- true

						return nil
					}
				})

				By("starting the handler to consume events", func() {
					// Set a context with a timeout to ensure the event handler isn't waiting forever
					var evCancel context.CancelFunc // Used to cancel the reading context
					evCtx, evCancel = context.WithTimeout(ctx, time.Second*15)

					for _, pID := range partitionIDs {
						_, err := hub.Receive(
							ctx,
							pID,
							eventHandler,
							eventhubs.ReceiveWithStartingOffset(persist.StartOfStream),
						)

						Expect(err).ToNot(HaveOccurred())
					}

					DeferCleanup(func() {
						evCancel()
					})
				})

				By("sending an event", func() {
					event = e2ece.NewHelloEvent(f)

					j := e2ece.RunEventSender(f.KubeClient, ns, tgtURL.String(), event)
					apps.WaitForCompletion(f.KubeClient, j)
				})

				By("waiting for the event to be received", func() {
					// don't exit till event is received by handler or times out
					select {
					case <-eventReceivedChannel:
					case <-evCtx.Done():
						framework.FailfWithOffset(2, "timed out while waiting for event")
					}
				})

				By("verifying the sent event", func() {
					Expect(len(payload)).To(BeNumerically(">", 0))
					var ce cloudevents.Event

					err := json.Unmarshal(payload, &ce)
					Expect(err).ToNot(HaveOccurred())

					Expect(ce.Data()).To(ContainSubstring(string(event.Data())))
					Expect(ce.ID()).To(Equal(event.ID()))
					Expect(ce.Type()).To(Equal(event.Type()))
					Expect(ce.Subject()).To(Equal(event.Subject()))
				})
			})
		})

		When("the spec contains default settings and the CloudEvent Context disabled", func() {
			var tgtURL *url.URL

			BeforeEach(func() {
				By("creating an AzureEventHubsTarget object", func() {
					tgt, err := createTarget(tgtClient, ns, "test-",
						withServicePrincipal(),
						withDiscardCEContextEnabled(),
						withEventHubID(subscriptionID, rg, ns, ns))

					Expect(err).ToNot(HaveOccurred())

					tgt = ducktypes.WaitUntilReady(f.DynamicClient, tgt)
					tgtURL = ducktypes.Address(tgt)
					Expect(tgtURL).ToNot(BeNil())
				})
			})

			It("receives an event on the Event Hub", func() {
				var event *cloudevents.Event
				var partitionIDs []string
				var eventHandler eventhubs.Handler
				var evCtx context.Context // Used to set a timeout for reading events
				var payload []byte

				eventReceivedChannel := make(chan bool)

				By("retrieving Event Hub partition details", func() {
					info, err := hub.GetRuntimeInformation(ctx)
					Expect(err).NotTo(HaveOccurred())

					partitionIDs = info.PartitionIDs
					Expect(len(partitionIDs)).To(BeNumerically(">", 0))
				})

				By("setting up a handler to verify the received event", func() {
					eventHandler = func(ctx context.Context, ev *eventhubs.Event) error {
						payload = ev.Data

						// Pass the bool to the channel to terminate the receiver
						eventReceivedChannel <- true

						return nil
					}
				})

				By("starting the handler to consume events", func() {
					// Set a context with a timeout to ensure the event handler isn't waiting forever
					var evCancel context.CancelFunc // Used to cancel the reading context
					evCtx, evCancel = context.WithTimeout(ctx, time.Second*15)

					for _, pID := range partitionIDs {
						_, err := hub.Receive(
							ctx,
							pID,
							eventHandler,
							eventhubs.ReceiveWithStartingOffset(persist.StartOfStream),
						)

						Expect(err).ToNot(HaveOccurred())
					}

					DeferCleanup(func() {
						evCancel()
					})
				})

				By("sending an event", func() {
					event = e2ece.NewHelloEvent(f)

					j := e2ece.RunEventSender(f.KubeClient, ns, tgtURL.String(), event)
					apps.WaitForCompletion(f.KubeClient, j)
				})

				By("waiting for the event to be received", func() {
					// don't exit till event is received by handler or times out
					select {
					case <-eventReceivedChannel:
					case <-evCtx.Done():
						framework.FailfWithOffset(2, "timed out while waiting for event")
					}
				})

				By("verifying the sent event", func() {
					Expect(len(payload)).To(BeNumerically(">", 0))
					Expect(payload).To(Equal(string(event.Data())))
				})
			})
		})
	})

	When("a client creates a target with invalid details", func() {
		subscriptionID := "00000000-0000-0000-0000-000000000000"

		It("should reject the creation of the target", func() {
			By("setting an invalid eventHubID", func() {
				invalidEventHubID := "im-just-a-random-string!"

				_, err := createTarget(tgtClient, ns, "test-invalid-eventhubid",
					withServicePrincipal(),
					withEventHubID(subscriptionID, ns, ns, invalidEventHubID))

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.eventHubID: Invalid value: "))
			})

			By("omitting the eventHubID", func() {
				_, err := createTarget(tgtClient, ns, "test-missing-eventhubid",
					withServicePrincipal())

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.eventHubID: Required value"))
			})

			By("omitting the credentials", func() {
				_, err := createTarget(tgtClient, ns, "test-missing-service-principal",
					withEventHubID(subscriptionID, ns, ns, ns))

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.auth: Required value"))
			})
		})
	})
})

// createTarget creates an AzureEventHubsTarget object initialized with the given options.
func createTarget(tgtClient dynamic.ResourceInterface, namespace, namePrefix string, opts ...targetOption) (*unstructured.Unstructured, error) {
	tgt := &unstructured.Unstructured{}
	tgt.SetAPIVersion(targetAPIVersion.String())
	tgt.SetKind(targetKind)
	tgt.SetNamespace(namespace)
	tgt.SetGenerateName(namePrefix)

	for _, opt := range opts {
		opt(tgt)
	}

	return tgtClient.Create(context.Background(), tgt, metav1.CreateOptions{})
}

type targetOption func(*unstructured.Unstructured)

// withServicePrincipal will create the secret and service principal based on the azure environment variables
func withServicePrincipal() targetOption {
	credsMap := map[string]interface{}{
		"tenantID":     map[string]interface{}{"value": os.Getenv("AZURE_TENANT_ID")},
		"clientID":     map[string]interface{}{"value": os.Getenv("AZURE_CLIENT_ID")},
		"clientSecret": map[string]interface{}{"value": os.Getenv("AZURE_CLIENT_SECRET")},
	}

	return func(tgt *unstructured.Unstructured) {
		if err := unstructured.SetNestedMap(tgt.Object, credsMap, "spec", "auth", "servicePrincipal"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.auth.servicePrincipal field: %s", err)
		}
	}
}

func withEventHubID(subscriptionID, resourceGroup, eventHubNS, eventHub string) targetOption {
	eventHubID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.EventHub/namespaces/%s/eventHubs/%s", subscriptionID, resourceGroup, eventHubNS, eventHub)

	return func(tgt *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(tgt.Object, eventHubID, "spec", "eventHubID"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.eventHubID field: %s", err)
		}
	}
}

func withDiscardCEContextEnabled() targetOption {
	return func(tgt *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(tgt.Object, true, "spec", "discardCloudEventContext"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.discardCloudEventContext field: %s", err)
		}
	}
}
