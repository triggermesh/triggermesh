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

package eventgrid

import (
	"fmt"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid/eventgridapi"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub/eventhubapi"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources/resourcesapi"
	"github.com/Azure/go-autorest/autorest"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/auth/azure"
)

// SystemTopicsClient wraps the eventgridapi.SystemTopicsClientAPI interface.
type SystemTopicsClient interface {
	eventgridapi.SystemTopicsClientAPI
	BaseClient() autorest.Client
	ConcreteClient() eventgrid.SystemTopicsClient
}

var _ SystemTopicsClient = (*SystemTopicsClientImpl)(nil)

// SystemTopicsClientImpl implements SystemTopicsClient with a concrete
// eventgrid.SystemTopicsClientImpl.
type SystemTopicsClientImpl struct {
	eventgrid.SystemTopicsClient
}

// BaseClient implements SystemTopicsClient.
func (c *SystemTopicsClientImpl) BaseClient() autorest.Client {
	return c.Client
}

// ConcreteClient implements SystemTopicsClient.
func (c *SystemTopicsClientImpl) ConcreteClient() eventgrid.SystemTopicsClient {
	return c.SystemTopicsClient
}

// ProvidersClient is an alias for the ProvidersClientAPI interface.
type ProvidersClient = resourcesapi.ProvidersClientAPI

// ResourceGroupsClient is an alias for the GroupsClientAPI interface.
type ResourceGroupsClient = resourcesapi.GroupsClientAPI

// EventSubscriptionsClient wraps the eventgridapi.SystemTopicEventSubscriptionsClientAPI interface.
type EventSubscriptionsClient interface {
	eventgridapi.SystemTopicEventSubscriptionsClientAPI
	BaseClient() autorest.Client
	ConcreteClient() eventgrid.SystemTopicEventSubscriptionsClient
}

var _ EventSubscriptionsClient = (*EventSubscriptionsClientImpl)(nil)

// EventSubscriptionsClientImpl implements EventSubscriptionsClient with a concrete
// eventgrid.SystemTopicEventSubscriptionsClient.
type EventSubscriptionsClientImpl struct {
	eventgrid.SystemTopicEventSubscriptionsClient
}

// BaseClient implements EventSubscriptionsClient.
func (c *EventSubscriptionsClientImpl) BaseClient() autorest.Client {
	return c.Client
}

// ConcreteClient implements EventSubscriptionsClient.
func (c *EventSubscriptionsClientImpl) ConcreteClient() eventgrid.SystemTopicEventSubscriptionsClient {
	return c.SystemTopicEventSubscriptionsClient
}

// EventHubsClient is an alias for the EventHubsClientAPI interface.
type EventHubsClient = eventhubapi.EventHubsClientAPI

// ClientGetter can obtain clients for Azure Event Grid and Event Hubs APIs.
type ClientGetter interface {
	Get(*v1alpha1.AzureEventGridSource) (
		SystemTopicsClient,
		ProvidersClient,
		ResourceGroupsClient,
		EventSubscriptionsClient,
		EventHubsClient,
		error)
}

// NewClientGetter returns a ClientGetter for the given secrets getter.
func NewClientGetter(sg NamespacedSecretsGetter) *ClientGetterWithSecretGetter {
	return &ClientGetterWithSecretGetter{
		sg: sg,
	}
}

// NamespacedSecretsGetter returns a SecretInterface for the given namespace.
type NamespacedSecretsGetter func(namespace string) coreclientv1.SecretInterface

// ClientGetterWithSecretGetter gets Azure clients using static credentials
// retrieved using a Secret getter.
type ClientGetterWithSecretGetter struct {
	sg NamespacedSecretsGetter
}

// ClientGetterWithSecretGetter implements ClientGetter.
var _ ClientGetter = (*ClientGetterWithSecretGetter)(nil)

// Get implements ClientGetter.
func (g *ClientGetterWithSecretGetter) Get(src *v1alpha1.AzureEventGridSource) (
	SystemTopicsClient, ProvidersClient, ResourceGroupsClient, EventSubscriptionsClient, EventHubsClient, error) {

	authorizer, err := azure.NewAADAuthorizer(g.sg(src.Namespace), src.Spec.Auth.ServicePrincipal)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("retrieving Azure service principal credentials: %w", err)
	}

	sysTopicsCli := &SystemTopicsClientImpl{
		SystemTopicsClient: eventgrid.NewSystemTopicsClient(src.Spec.Scope.SubscriptionID),
	}
	sysTopicsCli.Authorizer = authorizer

	providersCli := resources.NewProvidersClient(src.Spec.Scope.SubscriptionID)
	providersCli.Authorizer = authorizer

	resGroupsCli := resources.NewGroupsClient(src.Spec.Scope.SubscriptionID)
	resGroupsCli.Authorizer = authorizer

	eventSubsCli := &EventSubscriptionsClientImpl{
		SystemTopicEventSubscriptionsClient: eventgrid.NewSystemTopicEventSubscriptionsClient(src.Spec.Scope.SubscriptionID),
	}
	eventSubsCli.Authorizer = authorizer

	eventHubsCli := eventhub.NewEventHubsClient(src.Spec.Scope.SubscriptionID)
	eventHubsCli.Authorizer = authorizer

	return sysTopicsCli, providersCli, resGroupsCli, eventSubsCli, eventHubsCli, nil
}

// ClientGetterFunc allows the use of ordinary functions as ClientGetter.
type ClientGetterFunc func(*v1alpha1.AzureEventGridSource) (
	SystemTopicsClient, ProvidersClient, ResourceGroupsClient, EventSubscriptionsClient, EventHubsClient, error)

// ClientGetterFunc implements ClientGetter.
var _ ClientGetter = (ClientGetterFunc)(nil)

// Get implements ClientGetter.
func (f ClientGetterFunc) Get(src *v1alpha1.AzureEventGridSource) (
	SystemTopicsClient, ProvidersClient, ResourceGroupsClient, EventSubscriptionsClient, EventHubsClient, error) {

	return f(src)
}
