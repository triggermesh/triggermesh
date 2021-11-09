/*
Copyright 2021 TriggerMesh Inc.

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

package servicebus

import (
	"errors"
	"fmt"
	"strings"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/Azure/azure-amqp-common-go/v3/aad"
	amqpauth "github.com/Azure/azure-amqp-common-go/v3/auth"
	servicebus "github.com/Azure/azure-service-bus-go"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/azure/auth"
)

// ClientGetter can obtain a client for an Azure Service Bus namespace.
type ClientGetter interface {
	Get(*v1alpha1.AzureServiceBusTopicSource) (*Namespace, error)
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
func (g *ClientGetterWithSecretGetter) Get(src *v1alpha1.AzureServiceBusTopicSource) (*Namespace, error) {
	azureEnv := &azure.PublicCloud

	secretCli := g.sg(src.Namespace)

	var tokenProvider amqpauth.TokenProvider

	switch {
	case src.Spec.Auth.ServicePrincipal != nil:
		tpCtor, err := auth.NewAADTokenProviderCtor(secretCli, src.Spec.Auth.ServicePrincipal)
		if err != nil {
			return nil, fmt.Errorf("obtaining constructor for an AAD token provider: %w", err)
		}

		aadTokenProvider, err := tpCtor(
			aad.JWTProviderWithAzureEnvironment(azureEnv),
			// aad.NewJWTProvider configures the TokenProvider to
			// target Event Hubs by default, but this is not what
			// we want here.
			aad.JWTProviderWithResourceURI(azureEnv.ResourceIdentifiers.ServiceBus),
		)
		if err != nil {
			// TODO(antoine): Fix returned error upstream.
			//
			// As of v3.2.1, the Azure AMQP module attempts to refresh AAD tokens prior to returning a
			// TokenProvider but, in case of failure, returns an untyped and unformatted error (with
			// typos!): https://github.com/Azure/azure-amqp-common-go/blob/v3.2.1/aad/jwt.go#L160-L162
			//
			// Example:
			//   "failed to refersh token: &{{{  0 0 0  } 0xc001d4a240 {{https  <nil> login.microsoftonline.com
			//   /00000000-0000-0000-0000-000000000000  false   } {https  <nil> login.microsoftonline.com
			//   /00000000-0000-0000-0000-000000000000/oauth2/authorize  false api-version=1.0  } {https
			//   <nil> login.microsoftonline.com /00000000-0000-0000-0000-000000000000/oauth2/token
			//   false api-version=1.0  } {https  <nil>
			//   login.microsoftonline.com /00000000-0000-0000-0000-000000000000/oauth2/devicecode  false
			//   api-version=1.0  }} 00000000-0000-0000-0000-000000000000 https://servicebus.azure.net/ true
			//   300000000000} 0xc001f59380 0xc0029ac8a0 <nil> [] 0}"
			//
			// This error contains unique elements such as pointer addresses, and therefore needs to be
			// sanitized to avoid causing an infinite loop of reconciliations when written to an API
			// object's status.
			if strings.Contains(err.Error(), "failed to refersh token") { // (!) the typo is not a mistake
				err = errors.New("failed to refresh Service Principal token. The provided secret is " +
					"either invalid or expired")
			}

			return nil, auth.NewFatalCredentialsError(fmt.Errorf("constructing AAD token provider: %w", err))
		}

		tokenProvider = aadTokenProvider

	case src.Spec.Auth.SASToken != nil:
		tpCtor, err := auth.NewSASTokenProviderCtor(secretCli, src.Spec.Auth.SASToken)
		if err != nil {
			return nil, fmt.Errorf("obtaining constructor for a SAS token provider: %w", err)
		}

		sasTokenProvider, err := tpCtor()
		if err != nil {
			return nil, auth.NewFatalCredentialsError(fmt.Errorf("constructing SAS token provider: %w", err))
		}

		tokenProvider = sasTokenProvider
	}

	if tokenProvider == nil {
		return nil, errors.New("no supported auth method was provided")
	}

	return servicebus.NewNamespace(
		withName(src.Spec.TopicID.Namespace),
		withAzureEnvironment(azureEnv),
		servicebus.NamespaceWithTokenProvider(tokenProvider),
	)
}

// ClientGetterFunc allows the use of ordinary functions as ClientGetter.
type ClientGetterFunc func(*v1alpha1.AzureServiceBusTopicSource) (*Namespace, error)

// ClientGetterFunc implements ClientGetter.
var _ ClientGetter = (ClientGetterFunc)(nil)

// Get implements ClientGetter.
func (f ClientGetterFunc) Get(src *v1alpha1.AzureServiceBusTopicSource) (*Namespace, error) {
	return f(src)
}
