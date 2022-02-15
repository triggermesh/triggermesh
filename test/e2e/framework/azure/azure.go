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
	"math/rand"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// Package azure contains helpers for interacting with Azure and standing up prerequisite services

const E2EInstanceTagKey = "e2e_instance"

// CreateResourceGroup will create the resource group containing all of the eventhub components.
func CreateResourceGroup(ctx context.Context, subscriptionID, name, region string) armresources.ResourceGroup {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(1, "Unable to authenticate: %s", err)
	}

	rgClient := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)

	rg, err := rgClient.CreateOrUpdate(ctx, name, armresources.ResourceGroup{
		Location: to.StringPtr(region),
		Tags:     map[string]*string{E2EInstanceTagKey: to.StringPtr(name)},
	}, nil)

	if err != nil {
		framework.FailfWithOffset(1, "Unable to create resource group: %s", err)
	}

	return rg.ResourceGroup
}

// DeleteResourceGroup will delete everything under it allowing for easy cleanup
func DeleteResourceGroup(ctx context.Context, subscriptionID, name string) armresources.ResourceGroupsClientDeletePollerResponse {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(1, "Unable to authenticate: %s", err)
	}

	rgClient := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)

	resp, err := rgClient.BeginDelete(ctx, name, nil)
	if err != nil {
		framework.FailfWithOffset(1, "Resource group deletion failed: %s", err)
	}

	return resp
}

// WaitForFutureDeletion will wait on the resource to be deleted before continuing
func WaitForFutureDeletion(ctx context.Context, subscriptionID string, future armresources.ResourceGroupsClientDeletePollerResponse) {
	_, err := future.PollUntilDone(ctx, time.Second*30)
	if err != nil {
		framework.FailfWithOffset(1, "Resource group deletion failed: %s", err)
	}
}

// CreateStorageAccountCommon will create an azure storage account for both blob and queue storage tests
func CreateStorageAccountCommon(ctx context.Context, cli *armstorage.AccountsClient, name, rgName, region string, isBlob bool) armstorage.Account {
	storageParams := armstorage.AccountCreateParameters{
		Kind:     armstorage.KindStorage.ToPtr(),
		Location: &region,
		SKU: &armstorage.SKU{
			Name: armstorage.SKUNameStandardRAGRS.ToPtr(),
			Tier: armstorage.SKUTierStandard.ToPtr(),
		},
		Identity: &armstorage.Identity{
			Type: armstorage.IdentityTypeNone.ToPtr(),
		},
		Properties: &armstorage.AccountPropertiesCreateParameters{},
	}

	// Storage blob requires the access tier to be set and publicly available
	if isBlob {
		storageParams.Kind = armstorage.KindBlobStorage.ToPtr()
		storageParams.Properties = &armstorage.AccountPropertiesCreateParameters{
			AccessTier:            armstorage.AccessTierHot.ToPtr(),
			AllowBlobPublicAccess: to.BoolPtr(true),
		}
	}

	resp, err := cli.BeginCreate(ctx, rgName, name, storageParams, nil)

	if err != nil {
		framework.FailfWithOffset(3, "unable to create storage account: %s", err)
		return armstorage.Account{}
	}

	newSaClient, err := resp.PollUntilDone(ctx, time.Second*30)
	if err != nil {
		framework.FailfWithOffset(3, "unable to complete storage account creation: %s", err)
		return armstorage.Account{}
	}

	return newSaClient.Account
}

// randAlphanumString returns a random string of the given length containing
// only lowercase alphanumeric characters.
// It is useful to help in generating names for Azure resources.
func randAlphanumString(n int) string {
	const alphanumCharset = "0123456789abcdefghijklmnopqrstuvwxyz"

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, n)
	for i := range b {
		b[i] = alphanumCharset[r.Intn(len(alphanumCharset))]
	}
	return string(b)
}
