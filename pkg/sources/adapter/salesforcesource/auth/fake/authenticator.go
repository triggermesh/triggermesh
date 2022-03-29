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

package fake

import (
	"errors"

	"github.com/triggermesh/triggermesh/pkg/sources/adapter/salesforcesource/auth"
)

// Authenticator is a test oriented fake Authenticator.
type Authenticator struct {
	defaultCredentials auth.Credentials
}

var _ auth.Authenticator = (*Authenticator)(nil)

// NewFakeAuthenticator creates a Fake authenticator for Bayeux/Salesforce.
func NewFakeAuthenticator(defaultCredentials auth.Credentials) auth.Authenticator {
	return &Authenticator{
		defaultCredentials: defaultCredentials,
	}
}

// NewCredentials generates a new set of credentials.
func (a *Authenticator) NewCredentials() (*auth.Credentials, error) {
	return &a.defaultCredentials, nil
}

// RefreshCredentials renews credentials.
func (a *Authenticator) RefreshCredentials() (*auth.Credentials, error) {
	return nil, errors.New("not implemented")
}

// CreateOrRenewCredentials will always create a new set of credentials.
func (a *Authenticator) CreateOrRenewCredentials() (*auth.Credentials, error) {
	return a.NewCredentials()
}
