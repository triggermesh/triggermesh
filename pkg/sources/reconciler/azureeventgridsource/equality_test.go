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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"

	"knative.dev/pkg/logging"

	azureeventgrid "github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
)

func TestEqualEventSubscription(t *testing.T) {
	testCases := map[string]struct {
		x, y        azureeventgrid.EventSubscription
		expectEqual bool
	}{
		"Equal when only ignored fields differ": {
			x: azureeventgrid.EventSubscription{
				ID:   to.Ptr("x-id"),
				Name: to.Ptr("x-name"),
				Type: to.Ptr("x-type"),

				EventSubscriptionProperties: &azureeventgrid.EventSubscriptionProperties{
					Topic:             to.Ptr("x-topic"),
					ProvisioningState: "x-provisioningstate",
				},
			},
			y: azureeventgrid.EventSubscription{
				ID:   to.Ptr("y-id"),
				Name: to.Ptr("y-name"),
				Type: to.Ptr("y-type"),

				EventSubscriptionProperties: &azureeventgrid.EventSubscriptionProperties{
					Topic:             to.Ptr("y-topic"),
					ProvisioningState: "y-provisioningstate",
				},
			},
			expectEqual: true,
		},

		"Not equal when event destinations differ": {
			x: azureeventgrid.EventSubscription{
				EventSubscriptionProperties: &azureeventgrid.EventSubscriptionProperties{
					Destination: makeEventSubscriptionDestination("x-event-hub"),
				},
			},
			y: azureeventgrid.EventSubscription{
				EventSubscriptionProperties: &azureeventgrid.EventSubscriptionProperties{
					Destination: makeEventSubscriptionDestination("y-event-hub"),
				},
			},
			expectEqual: false,
		},

		"Not equal when event types differ": {
			x: azureeventgrid.EventSubscription{
				EventSubscriptionProperties: &azureeventgrid.EventSubscriptionProperties{
					Filter: makeEventSubscriptionEventFilter([]string{"type1", "type2"}),
				},
			},
			y: azureeventgrid.EventSubscription{
				EventSubscriptionProperties: &azureeventgrid.EventSubscriptionProperties{
					Filter: makeEventSubscriptionEventFilter([]string{"type2", "type3"}),
				},
			},
			expectEqual: false,
		},

		"Equal when one does not declare event types": {
			x: azureeventgrid.EventSubscription{
				EventSubscriptionProperties: &azureeventgrid.EventSubscriptionProperties{
					Filter: makeEventSubscriptionEventFilter([]string{"type1", "type2"}),
				},
			},
			y: azureeventgrid.EventSubscription{
				EventSubscriptionProperties: &azureeventgrid.EventSubscriptionProperties{
					Filter: makeEventSubscriptionEventFilter(nil),
				},
			},
			expectEqual: true,
		},
	}

	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())

	for n, tc := range testCases {
		//nolint:scopelint
		t.Run(n, func(t *testing.T) {
			equal := equalEventSubscription(ctx, tc.x, tc.y)
			assert.Equal(t, tc.expectEqual, equal)
		})
	}
}

// makeEventSubscriptionDestination generates an Event Grid subscription
// destination targetting an Event Hubs instance.
func makeEventSubscriptionDestination(eventHubID string) azureeventgrid.EventHubEventSubscriptionDestination {
	return azureeventgrid.EventHubEventSubscriptionDestination{
		EndpointType: azureeventgrid.EndpointTypeEventHub,
		EventHubEventSubscriptionDestinationProperties: &azureeventgrid.EventHubEventSubscriptionDestinationProperties{
			ResourceID: &eventHubID,
		},
	}
}

// makeEventSubscriptionEventFilter generates an Event Grid subscription filter
// matching the given event types.
func makeEventSubscriptionEventFilter(eventTypes []string) *azureeventgrid.EventSubscriptionFilter {
	return &azureeventgrid.EventSubscriptionFilter{
		IncludedEventTypes: &eventTypes,
	}
}
