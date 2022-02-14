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
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

const (
	AzureBlobStorageURL = ".blob.core.windows.net/"
)

// CreateBlobStorageAccount creates an Azure storage account in the given
// resource group and region.
// Because storage account names are globally unique, this helper generates a
// random name to reduce the risk of running into naming conflicts.
func CreateBlobStorageAccount(ctx context.Context, cli *armstorage.AccountsClient, rgName, region string) armstorage.Account {
	const maxStorAccNameLen = 24
	const stoAccNamePrefix = "tme2e"

	name := stoAccNamePrefix + randAlphanumString(maxStorAccNameLen-len(stoAccNamePrefix))

	return CreateStorageAccountCommon(ctx, cli, name, rgName, region, true)
}

// CreateBlobContainer will create a new blob storage container
func CreateBlobContainer(ctx context.Context, rg, saName, subscriptionID, name string) armstorage.BlobContainer {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to authenticate: %s", err)
	}

	client := armstorage.NewBlobContainersClient(subscriptionID, cred, nil)

	resp, err := client.Create(ctx, rg, saName, name, armstorage.BlobContainer{}, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to create blob container: %s", err)
	}

	return resp.BlobContainer
}

// UploadBlob will upload a new chunk of data to the blob storage
func UploadBlob(ctx context.Context, containerName, saName, name, data string) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to authenticate: %s", err)
	}

	url := "https://" + saName + AzureBlobStorageURL + containerName
	containerClient, err := azblob.NewContainerClient(url, cred, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to obtain blob client: %s", err)
	}

	blobClient := containerClient.NewBlockBlobClient(name)
	rs := ReadSeekCloser(strings.NewReader(data))

	_, err = blobClient.Upload(ctx, rs, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to upload payload: %s", err)
	}
}

// DeleteBlob will delete the blob located at the name location
func DeleteBlob(ctx context.Context, containerName, saName, name string) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to authenticate: %s", err)
	}

	url := "https://" + saName + AzureBlobStorageURL + containerName
	containerClient, err := azblob.NewContainerClient(url, cred, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to obtain blob client: %s", err)
	}

	blobClient := containerClient.NewBlockBlobClient(name)
	_, err = blobClient.Delete(ctx, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to delete blob: %s", err)
	}
}

// ReadSeekCloser implements a closer with Seek, Read, and Close
func ReadSeekCloser(r *strings.Reader) readSeekCloser {
	return readSeekCloser{r}
}

type readSeekCloser struct {
	*strings.Reader
}

func (readSeekCloser) Close() error { return nil }
