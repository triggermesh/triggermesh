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
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
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
func DeleteResourceGroup(ctx context.Context, subscriptionID, name string) armresources.ResourceGroupsDeletePollerResponse {
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
func WaitForFutureDeletion(ctx context.Context, subscriptionID string, future armresources.ResourceGroupsDeletePollerResponse) {
	_, err := future.PollUntilDone(ctx, time.Second*30)
	if err != nil {
		framework.FailfWithOffset(1, "Resource group deletion failed: %s", err)
	}
}
