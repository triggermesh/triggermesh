/*
Copyright (c) 2022 TriggerMesh Inc.

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
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-storage-queue-go/azqueue"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateStorageAccountsClient will create the storage account client
func CreateStorageAccountsClient(subscriptionID string) *armstorage.StorageAccountsClient {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(3, "unable to authenticate: %s", err)
	}

	saClient := armstorage.NewStorageAccountsClient(subscriptionID, cred, nil)

	return saClient
}

// CreateStorageAccount will create the storage account
func CreateStorageAccount(ctx context.Context, cli *armstorage.StorageAccountsClient, name, rgName, region string) error {
	resp, err := cli.BeginCreate(ctx, rgName, name, armstorage.StorageAccountCreateParameters{
		Kind:     armstorage.KindStorage.ToPtr(),
		Location: &region,
		SKU: &armstorage.SKU{
			Name: armstorage.SKUNameStandardRAGRS.ToPtr(),
			Tier: armstorage.SKUTierStandard.ToPtr(),
		},
		Identity: &armstorage.Identity{
			Type: armstorage.IdentityTypeNone.ToPtr(),
		},
		Properties: &armstorage.StorageAccountPropertiesCreateParameters{},
	}, nil)

	if err != nil {
		framework.FailfWithOffset(3, "unable to create storage account: %s", err)
		return err
	}

	_, err = resp.PollUntilDone(ctx, time.Second*30)
	if err != nil {
		framework.FailfWithOffset(3, "unable to complete storage account creation: %s", err)
		return err
	}

	return nil
}

// CreateQueueStorage will create a queue storage message url
func CreateQueueStorage(ctx context.Context, name, accountName string, accountKey string) *azqueue.MessagesURL {
	credential, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		framework.FailfWithOffset(3, "azqueue.NewSharedKeyCredential failed: ", err)
	}

	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})

	urlRef, err := url.Parse(fmt.Sprintf("https://%s.queue.core.windows.net", accountName))

	if err != nil {
		framework.FailfWithOffset(3, "url.Parse failed: ", err)
	}

	serviceURL := azqueue.NewServiceURL(*urlRef, p)

	// Create a Queue
	_, err = serviceURL.NewQueueURL(name).Create(ctx, azqueue.Metadata{})
	if err != nil {
		framework.FailfWithOffset(3, "error creating queue: ", err)
	}

	queueURL := serviceURL.NewQueueURL(name)
	messagesURL := queueURL.NewMessagesURL()

	return &messagesURL

}

// GetStorageAccountKey will return the storage account keys
func GetStorageAccountKey(ctx context.Context, cli *armstorage.StorageAccountsClient, name, rgName string) (armstorage.StorageAccountsListKeysResponse, error) {
	return cli.ListKeys(ctx, rgName, name, &armstorage.StorageAccountsListKeysOptions{})

}
