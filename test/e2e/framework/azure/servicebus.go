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

package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
	sv "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	svadmin "github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/admin"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateServiceBusNamespaceClient will create the servicebus client
func CreateServiceBusNamespaceClient(ctx context.Context, subscriptionID, region string) *servicebus.NamespacesClient {
	nsClient := servicebus.NewNamespacesClient(subscriptionID)

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		framework.FailfWithOffset(3, "unable to create authorizer: %s", err)
		return nil
	}

	nsClient.Authorizer = authorizer

	return &nsClient
}

func CreateServiceBusNamespace(ctx context.Context, cli servicebus.NamespacesClient, rgName string, nsName string, region string) error {
	// create the servicebus namespace
	nsFuture, err := cli.CreateOrUpdate(ctx, rgName, nsName, servicebus.SBNamespace{Location: to.StringPtr(region)})
	if err != nil {
		framework.FailfWithOffset(3, "unable to create servicebus namespace: %s", err)
		return err
	}

	// Wait for the namespace to be created before creating the topic
	err = nsFuture.WaitForCompletionRef(ctx, cli.Client)
	if err != nil {
		framework.FailfWithOffset(3, "unable to complete servicebus namespace creation: %s", err)
		return err
	}

	_, err = nsFuture.Result(cli)
	if err != nil {
		framework.FailfWithOffset(3, "servicebus namespace creation failed: %s", err)
		return err
	}

	return nil
}

// CreateClient will create a servicebus client.
func CreateClient(ctx context.Context, region string, name string, nsCli *servicebus.NamespacesClient) *sv.Client {
	keys, err := nsCli.ListKeys(ctx, name, name, "RootManageSharedAccessKey")
	if err != nil {
		framework.FailfWithOffset(3, "unable to obtain the connection string: %s", err)
		return nil
	}

	// Take the namespace connection string, and add the specific servicehub
	connectionString := *keys.PrimaryConnectionString + ";EntityPath=" + name
	client, err := sv.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		framework.FailfWithOffset(3, "creating client from connection string: %s", err)
		return nil
	}

	return client
}

// CreateAdminClient will create a servicebus admin client.
func CreateAdminClient(ctx context.Context, region string, name string, nsCli *servicebus.NamespacesClient) *svadmin.Client {
	keys, err := nsCli.ListKeys(ctx, name, name, "RootManageSharedAccessKey")
	if err != nil {
		framework.FailfWithOffset(3, "unable to obtain the connection string: %s", err)
		return nil
	}

	// Take the namespace connection string, and add the specific servicehub
	connectionString := *keys.PrimaryConnectionString + ";EntityPath=" + name
	client, err := svadmin.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		framework.FailfWithOffset(3, "creating admin client from connection string: %s", err)
		return nil
	}

	return client
}
