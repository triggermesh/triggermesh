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
	"errors"
	"fmt"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/secret"
)

// Authorizer returns a new Authorizer for autorest-based Azure clients using
// the provided service principal authentication information.
func Authorizer(cli coreclientv1.SecretInterface, spAuth *v1alpha1.AzureServicePrincipal) (autorest.Authorizer, error) {
	if spAuth == nil {
		return nil, errors.New("servicePrincipal auth is undefined")
	}

	requestedSecrets, err := secret.NewGetter(cli).Get(
		spAuth.TenantID,
		spAuth.ClientID,
		spAuth.ClientSecret,
	)
	if err != nil {
		return nil, fmt.Errorf("getting auth secrets: %w", err)
	}

	tenantID, clientID, clientSecret := requestedSecrets[0], requestedSecrets[1], requestedSecrets[2]

	authSettings := auth.EnvironmentSettings{
		Values: map[string]string{
			auth.TenantID:     tenantID,
			auth.ClientID:     clientID,
			auth.ClientSecret: clientSecret,
			auth.Resource:     azure.PublicCloud.ResourceManagerEndpoint,
		},
		Environment: azure.PublicCloud,
	}

	authorizer, err := authSettings.GetAuthorizer()
	if err != nil {
		// GetAuthorizer returns an untyped error when it is unable to
		// obtain a non-empty value for any of the required auth settings.
		return nil, NewFatalCredentialsError(err)
	}

	return authorizer, nil
}
