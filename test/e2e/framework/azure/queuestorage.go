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
	"fmt"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-storage-queue-go/azqueue"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateStorageAccountsClient will create the storage account client
func CreateStorageAccountsClient(subscriptionID string) *armstorage.AccountsClient {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(3, "unable to authenticate: %s", err)
	}

	saClient := armstorage.NewAccountsClient(subscriptionID, cred, nil)

	return saClient
}

// CreateQueueStorageAccount provides a wrapper to support Queue storage test
func CreateQueueStorageAccount(ctx context.Context, cli *armstorage.AccountsClient, name, rgName, region string) armstorage.Account {
	return CreateStorageAccountCommon(ctx, cli, name, rgName, region, false)
}

// CreateQueueStorage will create a queue storage message url
func CreateQueueStorage(ctx context.Context, name, accountName string, accountKey string) *azqueue.MessagesURL {
	credential, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to obtain azqueue.NewSharedKeyCredential: %s", err)
	}

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})

	urlRef, err := url.Parse(fmt.Sprintf("https://%s.queue.core.windows.net", accountName))

	if err != nil {
		framework.FailfWithOffset(2, "Failed to parse url: %s", err)
	}

	serviceURL := azqueue.NewServiceURL(*urlRef, p)

	// Create a Queue
	_, err = serviceURL.NewQueueURL(name).Create(ctx, azqueue.Metadata{})
	if err != nil {
		framework.FailfWithOffset(2, "Error creating queue: %s", err)
	}

	queueURL := serviceURL.NewQueueURL(name)
	messagesURL := queueURL.NewMessagesURL()

	return &messagesURL

}

// GetStorageAccountKey will return the storage account keys
func GetStorageAccountKey(ctx context.Context, cli *armstorage.AccountsClient, name, rgName string) (armstorage.AccountsClientListKeysResponse, error) {
	return cli.ListKeys(ctx, rgName, name, &armstorage.AccountsClientListKeysOptions{})
}
