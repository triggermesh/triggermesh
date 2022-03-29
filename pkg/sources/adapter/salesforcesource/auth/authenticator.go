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

package auth

// Authenticator manages and stores Authentication Credentials
// for services.
type Authenticator interface {
	// CreateOrRenewCredentials is a best effort wrapper on credential
	// provisioning functions that can decide if it is better to create
	// new credentials or refresh using an existing token.
	CreateOrRenewCredentials() (*Credentials, error)
	// NewCredentials retrieve a new set of credentials.
	NewCredentials() (*Credentials, error)
	// RefreshCredentials uses credentials refresh tokens to create a new set of credentials.
	RefreshCredentials() (*Credentials, error)
}

// Credentials returned from Salesforce Auth.
type Credentials struct {
	Token        string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	InstanceURL  string `json:"instance_url"`
	ID           string `json:"id"`
	Signature    string `json:"signature"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
	CommunityURL string `json:"sfdc_community_url"`
	CommunityID  string `json:"sfdc_community_id"`
}
