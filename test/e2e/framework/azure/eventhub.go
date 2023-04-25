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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

func CreateEventHubNamespaceOnly(ctx context.Context, subscriptionID, name, region, rg string) *armeventhub.Eventhub {
	return CreateEventHubCommon(ctx, subscriptionID, name, region, rg, true)
}

func CreateEventHubComponents(ctx context.Context, subscriptionID, name, region, rg string) *armeventhub.Eventhub {
	return CreateEventHubCommon(ctx, subscriptionID, name, region, rg, false)
}

func CreateEventHubCommon(ctx context.Context, subscriptionID, name, region, rg string, omitHub bool) *armeventhub.Eventhub {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(1, "Unable to authenticate: %s", err)
	}

	nsClient, err := armeventhub.NewNamespacesClient(subscriptionID, cred, nil)
	if err != nil {
		framework.FailfWithOffset(1, "Failed to create Event Hubs namespaces client: %s", err)
	}

	ehClient, err := armeventhub.NewEventHubsClient(subscriptionID, cred, nil)
	if err != nil {
		framework.FailfWithOffset(1, "Failed to create Event Hubs client: %s", err)
	}

	// create the eventhubs namespace
	nsResp, err := nsClient.BeginCreateOrUpdate(ctx, rg, name, armeventhub.EHNamespace{
		Location: &region,
		Tags:     map[string]*string{E2EInstanceTagKey: to.Ptr(name)},
		Identity: &armeventhub.Identity{
			Type: to.Ptr(armeventhub.ManagedServiceIdentityTypeNone),
		},
		SKU: &armeventhub.SKU{
			Name:     to.Ptr(armeventhub.SKUNameBasic),
			Capacity: to.Ptr[int32](1),
			Tier:     to.Ptr(armeventhub.SKUTierBasic),
		},
	}, nil)

	if err != nil {
		framework.FailfWithOffset(1, "Unable to create eventhub namespace: %s", err)
	}

	if _, err = nsResp.PollUntilDone(ctx, pollOpts); err != nil {
		framework.FailfWithOffset(1, "Unable to create eventhub namespace: %s", err)
	}

	if !omitHub {
		ehResp, err := ehClient.CreateOrUpdate(ctx, rg, name, name, armeventhub.Eventhub{
			Properties: &armeventhub.Properties{
				MessageRetentionInDays: to.Ptr[int64](1),
				PartitionCount:         to.Ptr[int64](1),
			},
		}, nil)

		if err != nil {
			framework.FailfWithOffset(1, "Unable to create eventhub: %s", err)
			return nil
		}

		return &ehResp.Eventhub
	}

	return nil
}
