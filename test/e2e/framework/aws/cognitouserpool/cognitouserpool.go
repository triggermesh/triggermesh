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

// Package cognitouser contains helpers for AWS Cognito User Pools.
package cognitouser

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateUserPool creates a userpool named after the given framework.Framework.
func CreateUserPool(cc cognitoidentityprovideriface.CognitoIdentityProviderAPI, f *framework.Framework) string /*pool arn*/ {
	params := &cognitoidentityprovider.CreateUserPoolInput{
		PoolName: &f.UniqueName,
	}

	output, err := cc.CreateUserPool(params)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create userpool %q: %s", *params.PoolName, err)
	}

	return *output.UserPool.Arn
}

// CreateUser creates a new user in a user pool
func CreateUser(cc cognitoidentityprovideriface.CognitoIdentityProviderAPI, id string, username string) {
	params := &cognitoidentityprovider.AdminCreateUserInput{
		Username:   aws.String(username),
		UserPoolId: aws.String(id),
	}

	if _, err := cc.AdminCreateUser(params); err != nil {
		framework.FailfWithOffset(2, "Failed to create new user in pool: %s", err)
	}
}

// DeleteUserPool deletes a cognito userpool by pool id.
func DeleteUserPool(cc cognitoidentityprovideriface.CognitoIdentityProviderAPI, id string) {
	params := &cognitoidentityprovider.DeleteUserPoolInput{
		UserPoolId: aws.String(id),
	}

	if _, err := cc.DeleteUserPool(params); err != nil {
		framework.FailfWithOffset(2, "Failed to delete userpool %q: %s", *params.UserPoolId, err)
	}
}
