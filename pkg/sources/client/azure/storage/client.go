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

package storage

import (
	"fmt"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid/eventgridapi"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub/eventhubapi"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/auth/azure"
)

// EventSubscriptionsClient is an alias for the EventSubscriptionsClientAPI interface.
type EventSubscriptionsClient = eventgridapi.EventSubscriptionsClientAPI

// EventHubsClient is an alias for the EventHubsClientAPI interface.
type EventHubsClient = eventhubapi.EventHubsClientAPI

// ClientGetter can obtain clients for Azure Event Grid and Event Hubs APIs.
type ClientGetter interface {
	Get(*v1alpha1.AzureBlobStorageSource) (EventSubscriptionsClient, EventHubsClient, error)
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
func (g *ClientGetterWithSecretGetter) Get(src *v1alpha1.AzureBlobStorageSource) (EventSubscriptionsClient, EventHubsClient, error) {
	var authorizer autorest.Authorizer
	var err error

	if src.Spec.Auth.ServicePrincipal != nil {
		authorizer, err = azure.NewAADAuthorizer(g.sg(src.Namespace), src.Spec.Auth.ServicePrincipal)
		if err != nil {
			return nil, nil, fmt.Errorf("retrieving Azure service principal credentials: %w", err)
		}
	} else {
		// Use Azure AKS Managed Identity
		authorizer, err = auth.NewAuthorizerFromEnvironment()
		if err != nil {
			return nil, nil, fmt.Errorf("retrieving Azure AKS Managed Identity: %w", err)
		}
	}

	eventSubsCli := eventgrid.NewEventSubscriptionsClient(src.Spec.StorageAccountID.SubscriptionID)
	eventSubsCli.Authorizer = authorizer

	eventHubsCli := eventhub.NewEventHubsClient(src.Spec.StorageAccountID.SubscriptionID)
	eventHubsCli.Authorizer = authorizer

	return eventSubsCli, eventHubsCli, nil
}

// ClientGetterFunc allows the use of ordinary functions as ClientGetter.
type ClientGetterFunc func(*v1alpha1.AzureBlobStorageSource) (EventSubscriptionsClient, EventHubsClient, error)

// ClientGetterFunc implements ClientGetter.
var _ ClientGetter = (ClientGetterFunc)(nil)

// Get implements ClientGetter.
func (f ClientGetterFunc) Get(src *v1alpha1.AzureBlobStorageSource) (EventSubscriptionsClient, EventHubsClient, error) {
	return f(src)
}
