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

package auth

import (
	"fmt"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/Azure/azure-amqp-common-go/v3/aad"
	"github.com/Azure/azure-amqp-common-go/v3/conn"
	"github.com/Azure/azure-amqp-common-go/v3/sas"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/secret"
)

// AADTokenProviderCtor is a constructor for a AMQP TokenProvider for AAD
// authentication (Service Principal).
type AADTokenProviderCtor func(opts ...aad.JWTProviderOption) (*aad.TokenProvider, error)

// SASTokenProviderCtor is a constructor for a AMQP TokenProvider for SAS
// authentication (Secret Key, Connection String).
type SASTokenProviderCtor func(opts ...sas.TokenProviderOption) (*sas.TokenProvider, error)

// NewAADTokenProviderCtor returns a AADTokenProviderCtor with credentials
// pre-populated from the given Service Principal authentication information.
func NewAADTokenProviderCtor(cli coreclientv1.SecretInterface, spAuth *v1alpha1.AzureServicePrincipal) (AADTokenProviderCtor, error) {
	requestedSecrets, err := secret.NewGetter(cli).Get(
		spAuth.TenantID,
		spAuth.ClientID,
		spAuth.ClientSecret,
	)
	if err != nil {
		return nil, fmt.Errorf("getting auth secrets: %w", err)
	}

	tenantID, clientID, clientSecret := requestedSecrets[0], requestedSecrets[1], requestedSecrets[2]

	jwtProviderFromServicePrincipal := func(config *aad.TokenProviderConfiguration) error {
		config.TenantID = tenantID
		config.ClientID = clientID
		config.ClientSecret = clientSecret
		return nil
	}

	tpCtor := func(opts ...aad.JWTProviderOption) (*aad.TokenProvider, error) {
		return aad.NewJWTProvider(
			append(
				[]aad.JWTProviderOption{jwtProviderFromServicePrincipal},
				opts...,
			)...,
		)
	}

	return tpCtor, nil
}

// NewSASTokenProviderCtor returns a SASTokenProviderCtor with credentials
// pre-populated from the given SAS token authentication information.
func NewSASTokenProviderCtor(cli coreclientv1.SecretInterface, sasAuth *v1alpha1.AzureSASToken) (SASTokenProviderCtor, error) {
	requestedSecrets, err := secret.NewGetter(cli).Get(
		sasAuth.KeyName,
		sasAuth.KeyValue,
		sasAuth.ConnectionString,
	)
	if err != nil {
		return nil, fmt.Errorf("getting auth secrets: %w", err)
	}

	keyName, keyValue, connStr := requestedSecrets[0], requestedSecrets[1], requestedSecrets[2]

	if keyName == "" && keyValue == "" {
		parsedConn, err := conn.ParsedConnectionFromStr(connStr)
		if err != nil {
			return nil, err
		}

		keyName = parsedConn.KeyName
		keyValue = parsedConn.Key
	}

	tpCtor := func(opts ...sas.TokenProviderOption) (*sas.TokenProvider, error) {
		return sas.NewTokenProvider(
			append(
				[]sas.TokenProviderOption{sas.TokenProviderWithKey(keyName, keyValue)},
				opts...,
			)...,
		)
	}

	return tpCtor, nil
}
