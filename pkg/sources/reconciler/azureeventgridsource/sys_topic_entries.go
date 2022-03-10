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

package azureeventgridsource

import (
	"errors"
	"strconv"
	"strings"

	"knative.dev/pkg/controller"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

// topicType returns a suitable type for a system topic based on the provided
// provider and resource type.
// Topic type values represent the event sources that support the creation of
// system topics.
func topicType(provider, resourceType string) (*topicTypeEntry, error) {
	topicTypeResources, exists := allTopicTypeResourcesByProvider()[provider]
	if !exists {
		return nil, controller.NewPermanentError(errors.New("no supported system topic type for resource " +
			strconv.Quote(provider+"/"+resourceType)))
	}

	topicTypeRegionType, exists := topicTypeResources[resourceType]
	if !exists {
		return nil, controller.NewPermanentError(errors.New("no supported system topic type for resource " +
			strconv.Quote(provider+"/"+resourceType)))
	}

	return &topicTypeEntry{
		typ:        provider + "." + resourceType,
		regionType: topicTypeRegionType,
	}, nil
}

// topicTypeResourcesByProvider maps Azure providers to Azure resources
// supported by Event Grid system topics.
type topicTypeResourcesByProvider map[ /*provider*/ string]topicTypeRegionTypeByResource

// topicTypeRegionTypeByResource maps Azure resources supported by Event Grid
// system topics to their region type.
type topicTypeRegionTypeByResource map[ /*resource*/ string]systemTopicRegionType

// allTopicTypeResourcesByProvider returns all known supported Event Grid topic
// types indexed by Azure provider.
func allTopicTypeResourcesByProvider() topicTypeResourcesByProvider {
	return topicTypeResourcesByProvider{
		"microsoft.agfoodplatform": {
			"farmbeats": regionalResource,
		},
		"microsoft.apimanagement": {
			"service": regionalResource,
		},
		"microsoft.appconfiguration": {
			"configurationstores": regionalResource,
		},
		"microsoft.cache": {
			"redis": regionalResource,
		},
		"microsoft.communication": {
			"communicationservices": globalResource,
		},
		"microsoft.containerregistry": {
			"registries": regionalResource,
		},
		"microsoft.devices": {
			"iothubs": regionalResource,
		},
		"microsoft.eventhub": {
			"namespaces": regionalResource,
		},
		"microsoft.keyvault": {
			"vaults": regionalResource,
		},
		"microsoft.machinelearningservices": {
			"workspaces": regionalResource,
		},
		"microsoft.maps": {
			"accounts": globalResource,
		},
		"microsoft.media": {
			"mediaservices": regionalResource,
		},
		"microsoft.resources": {
			"subscriptions":  globalResource,
			"resourcegroups": globalResource,
		},
		"microsoft.servicebus": {
			"namespaces": regionalResource,
		},
		"microsoft.signalrservice": {
			"signalr": regionalResource,
		},
		"microsoft.storage": {
			"storageaccounts": regionalResource,
		},
		"microsoft.web": {
			"sites":       regionalResource,
			"serverfarms": regionalResource,
		},
	}
}

// topicTypeEntry associates a type of Event Grid system topic to its region type.
type topicTypeEntry struct {
	typ        string
	regionType systemTopicRegionType
}

// systemTopicRegionType
type systemTopicRegionType uint8

const (
	regionalResource systemTopicRegionType = iota
	globalResource
)

// providerAndResourceType returns sane values of Azure provider and resource
// type based on the given scope.
func providerAndResourceType(scope v1alpha1.AzureResourceID) (string /*provider*/, string /*resource type*/) {
	provider := strings.ToLower(scope.ResourceProvider)
	resourceType := strings.ToLower(scope.ResourceType)
	if provider == "" {
		provider = "microsoft.resources"

		resourceType = "subscriptions"
		if scope.ResourceGroup != "" {
			resourceType = "resourcegroups"
		}
	}

	return provider, resourceType
}
