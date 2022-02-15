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
	"time"

	eventhubs "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

func CreateEventHubNamespaceOnly(ctx context.Context, subscriptionID, name, region, rg string) *eventhubs.Hub {
	return CreateEventHubCommon(ctx, subscriptionID, name, region, rg, true)
}

func CreateEventHubComponents(ctx context.Context, subscriptionID, name, region, rg string) *eventhubs.Hub {
	return CreateEventHubCommon(ctx, subscriptionID, name, region, rg, false)
}

func CreateEventHubCommon(ctx context.Context, subscriptionID, name, region, rg string, omitHub bool) *eventhubs.Hub {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(1, "Unable to authenticate: %s", err)
	}

	nsClient := armeventhub.NewNamespacesClient(subscriptionID, cred, nil)
	ehClient := armeventhub.NewEventHubsClient(subscriptionID, cred, nil)

	// create the eventhubs namespace
	nsResp, err := nsClient.BeginCreateOrUpdate(ctx, rg, name, armeventhub.EHNamespace{
		Location: &region,
		Tags:     map[string]*string{E2EInstanceTagKey: to.StringPtr(name)},
		Identity: &armeventhub.Identity{
			Type: armeventhub.ManagedServiceIdentityTypeNone.ToPtr(),
		},
		SKU: &armeventhub.SKU{
			Name:     armeventhub.SKUNameBasic.ToPtr(),
			Capacity: to.Int32Ptr(1),
			Tier:     armeventhub.SKUTierBasic.ToPtr(),
		},
	}, nil)

	if err != nil {
		framework.FailfWithOffset(1, "Unable to create eventhub namespace: %s", err)
	}

	_, err = nsResp.PollUntilDone(ctx, time.Second*30)
	if err != nil {
		framework.FailfWithOffset(1, "Unable to create eventhub namespace: %s", err)
	}

	if !omitHub {
		ehResp, err := ehClient.CreateOrUpdate(ctx, rg, name, name, armeventhub.Eventhub{
			Properties: &armeventhub.Properties{
				MessageRetentionInDays: to.Int64Ptr(1),
				PartitionCount:         to.Int64Ptr(2),
			},
		}, nil)

		if err != nil {
			framework.FailfWithOffset(1, "Unable to create eventhub: %s", err)
			return nil
		}

		hub, err := eventhubs.NewHubWithNamespaceNameAndEnvironment(*ehResp.Name, name)
		if err != nil {
			framework.FailfWithOffset(1, "Unable to create eventhub client: %s", err)
			return nil
		}

		return hub
	}

	return nil
}
