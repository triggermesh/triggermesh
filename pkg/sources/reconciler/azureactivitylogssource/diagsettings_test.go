/*
Copyright 2021 TriggerMesh Inc.

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

package azureactivitylogssource

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"

	"k8s.io/client-go/tools/record"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	clients "github.com/triggermesh/triggermesh/pkg/sources/client/azure/insights"
)

var fakeLogCategories = []string{
	// voluntarily shuffled so we can test that the order of categories
	// does not influence comparisons of API payloads
	"FakeLogCategoryB",
	"FakeLogCategoryC",
	"FakeLogCategoryA",
}

const tSubscriptionID = "00000000-0000-0000-0000-000000000000"

var (
	tEventHubNamespaceID = v1alpha1.AzureResourceID{
		SubscriptionID:   tSubscriptionID,
		ResourceGroup:    "MyGroup",
		ResourceProvider: "Microsoft.EventHub",
		ResourceType:     "namespaces",
		ResourceName:     "MyNamespace",
	}

	tEventHubID = v1alpha1.AzureResourceID{
		SubscriptionID:   tSubscriptionID,
		ResourceGroup:    "MyGroup",
		ResourceProvider: "Microsoft.EventHub",
		Namespace:        "MyNamespace",
		ResourceType:     "eventhubs",
		ResourceName:     "MyEventHub",
	}
)

const (
	tEventHubsSASPolicy   = "MyPolicy"
	tEventHubsSASPolicyID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/MyGroup" +
		"/providers/Microsoft.EventHub/namespaces/MyNamespace/authorizationRules/MyPolicy"
)

func TestEnsureDiagnosticSettings(t *testing.T) {
	testCases := []struct {
		name           string
		cliReqRecorder clientsWithRequestRecorder
		categories     []string
		expectPayload  *insights.DiagnosticSettingsResource
		expectErr      bool
	}{{
		name:           "At least one valid log category selected",
		cliReqRecorder: mockClients,
		categories:     []string{"FakeLogCategoryA", "DoesNotExist"},
		expectPayload: makePayload([]logCategoryTuple{
			{"FakeLogCategoryA", true},
			{"FakeLogCategoryB", false},
			{"FakeLogCategoryC", false},
		}...),
		expectErr: false,
	}, {
		name:           "No log category selected",
		cliReqRecorder: mockClients,
		categories:     nil, // implicitly select all categories
		expectPayload: makePayload([]logCategoryTuple{
			{"FakeLogCategoryA", true},
			{"FakeLogCategoryB", true},
			{"FakeLogCategoryC", true},
		}...),
		expectErr: false,
	}, {
		name:           "Diagnostic settings are up-to-date",
		cliReqRecorder: existingDiagSettingsMockClients,
		categories:     nil, // implicitly select all categories
		expectPayload:  nil,
		expectErr:      false,
	}, {
		name:           "None of the selected log categories is valid",
		cliReqRecorder: mockClients,
		categories:     []string{"DoesNotExist", "DoesNotExistEither"},
		expectPayload:  nil,
		expectErr:      true,
	}, {
		name:           "Empty list of log categories",
		cliReqRecorder: mockClients,
		categories:     []string{},
		expectPayload:  nil,
		expectErr:      true,
	}, {
		name:           "Fail to list event categories",
		cliReqRecorder: failEventCategoriesListMockClients,
		expectPayload:  nil,
		expectErr:      true,
	}, {
		name:           "Fail to get current diagnostic settings",
		cliReqRecorder: failGetDiagnosticSettingsMockClients,
		expectPayload:  nil,
		expectErr:      true,
	}}

	for _, tc := range testCases {
		//nolint:scopelint
		t.Run(tc.name, func(t *testing.T) {
			reqRecorder := make(chan insights.DiagnosticSettingsResource)
			defer close(reqRecorder)

			r := &Reconciler{
				cg: staticClientGetter(tc.cliReqRecorder(reqRecorder)),
			}

			src := &v1alpha1.AzureActivityLogsSource{
				Spec: v1alpha1.AzureActivityLogsSourceSpec{
					SubscriptionID: tSubscriptionID,
					Destination: v1alpha1.AzureActivityLogsSourceDestination{
						EventHubs: v1alpha1.AzureActivityLogsSourceDestinationEventHubs{
							NamespaceID: tEventHubNamespaceID,
							HubName:     &tEventHubID.ResourceName,
							SASPolicy:   to.StringPtr(tEventHubsSASPolicy),
						},
					},
					Categories: tc.categories,
				},
			}

			ctx := v1alpha1.WithSource(testContext(t), src)

			returnedErr := make(chan error)
			go func() {
				returnedErr <- r.ensureDiagnosticSettings(ctx)
			}()

			var createUpdPayload *insights.DiagnosticSettingsResource
			var err error

			select {
			case req := <-reqRecorder:
				createUpdPayload = &req
				err = <-returnedErr

			case err = <-returnedErr:
				t.Log("Function returned without creating/updating diagnostic settings. Err:", err)
			}

			// note: log categories are unsorted inside the payload, so we
			// sort them inside the comparison function to avoid false positives.
			if diff := cmp.Diff(tc.expectPayload, createUpdPayload, cmpopts.SortSlices(lessLogSettings)); diff != "" {
				t.Error("Payload differs from expectation (-want, +got)\n" + diff)
			}

			assert.Equal(t, tc.expectErr, err != nil)
		})
	}
}

// expected contents: {string,bool}
type logCategoryTuple [2]interface{}

func makePayload(cats ...logCategoryTuple) *insights.DiagnosticSettingsResource {
	logs := make([]insights.LogSettings, len(cats))
	for i := range cats {
		category := cats[i][0].(string)
		enabled := cats[i][1].(bool)

		logs[i] = insights.LogSettings{
			Category: &category,
			Enabled:  &enabled,
		}
	}

	return &insights.DiagnosticSettingsResource{
		DiagnosticSettings: &insights.DiagnosticSettings{
			EventHubAuthorizationRuleID: to.StringPtr(tEventHubsSASPolicyID),
			EventHubName:                &tEventHubID.ResourceName,
			Logs:                        &logs,
		},
	}
}

func testContext(t *testing.T) context.Context {
	logger := logtesting.TestLogger(t)

	const eventRecorderBufferSize = 10
	eventRecorder := record.NewFakeRecorder(eventRecorderBufferSize)

	ctx := logging.WithLogger(context.Background(), logger)
	ctx = controller.WithEventRecorder(ctx, eventRecorder)

	return ctx
}

/*
   Mock Azure clients
*/

// staticClientGetter transforms the given client interfaces into a
// ClientGetter.
func staticClientGetter(ecCli clients.EventCategoriesClient, dsCli clients.DiagnosticSettingsClient) clients.ClientGetterFunc {
	return func(*v1alpha1.AzureActivityLogsSource) (clients.EventCategoriesClient, clients.DiagnosticSettingsClient, error) {
		return ecCli, dsCli, nil
	}
}

// requestRecorder records calls to DiagnosticSettingsClient.CreateOrUpdate.
// The channel remains empty in case of absence of call to that function.
type requestRecorder chan<- insights.DiagnosticSettingsResource

// clientsWithRequestRecorder returns a pair of Azure clients with a given
// requestRecorder.
type clientsWithRequestRecorder func(requestRecorder) (clients.EventCategoriesClient, clients.DiagnosticSettingsClient)

func mockClients(rr requestRecorder) (clients.EventCategoriesClient, clients.DiagnosticSettingsClient) {
	return &mockEventCategoriesClient{},
		&mockDiagnosticSettingsClient{rr: rr}
}
func existingDiagSettingsMockClients(rr requestRecorder) (clients.EventCategoriesClient, clients.DiagnosticSettingsClient) {
	return &mockEventCategoriesClient{},
		&mockDiagnosticSettingsClient{
			rr: rr,
			getResp: &insights.DiagnosticSettingsResource{
				DiagnosticSettings: &insights.DiagnosticSettings{
					EventHubAuthorizationRuleID: to.StringPtr(tEventHubsSASPolicyID),
					EventHubName:                &tEventHubID.ResourceName,
					// assume all categories are selected for simplicity
					Logs: &[]insights.LogSettings{{
						Category: to.StringPtr("FakeLogCategoryA"),
						Enabled:  to.BoolPtr(true),
					}, {
						Category: to.StringPtr("FakeLogCategoryB"),
						Enabled:  to.BoolPtr(true),
					}, {
						Category: to.StringPtr("FakeLogCategoryC"),
						Enabled:  to.BoolPtr(true),
					}},
				},
			},
		}
}
func failEventCategoriesListMockClients(rr requestRecorder) (clients.EventCategoriesClient, clients.DiagnosticSettingsClient) {
	return &mockEventCategoriesClient{listErr: errFake},
		&mockDiagnosticSettingsClient{rr: rr}
}
func failGetDiagnosticSettingsMockClients(rr requestRecorder) (clients.EventCategoriesClient, clients.DiagnosticSettingsClient) {
	return &mockEventCategoriesClient{},
		&mockDiagnosticSettingsClient{
			getErr: errFake,
			rr:     rr,
		}
}

type mockEventCategoriesClient struct {
	clients.EventCategoriesClient
	listErr error
}

// mockEventCategoriesClient implements clients.EventCategoriesClient.
var _ clients.EventCategoriesClient = (*mockEventCategoriesClient)(nil)

func (cli *mockEventCategoriesClient) List(context.Context) (insights.EventCategoryCollection, error) {
	var resp insights.EventCategoryCollection

	if err := cli.listErr; err != nil {
		return resp, err
	}

	val := make([]insights.LocalizableString, len(fakeLogCategories))
	resp.Value = &val

	for i := range fakeLogCategories {
		(*resp.Value)[i] = insights.LocalizableString{Value: &fakeLogCategories[i]}
	}

	return resp, nil
}

// mockDiagnosticSettingsClient implements clients.DiagnosticSettingsClient.
type mockDiagnosticSettingsClient struct {
	clients.DiagnosticSettingsClient

	getResp *insights.DiagnosticSettingsResource
	getErr  error

	createUpdErr error
	deleteErr    error

	rr requestRecorder
}

var _ clients.DiagnosticSettingsClient = (*mockDiagnosticSettingsClient)(nil)

func (cli *mockDiagnosticSettingsClient) Get(context.Context, string, string) (insights.DiagnosticSettingsResource, error) {
	resp := insights.DiagnosticSettingsResource{}
	if mockResp := cli.getResp; mockResp != nil {
		resp = *mockResp
	}

	return resp, cli.getErr
}

func (cli *mockDiagnosticSettingsClient) CreateOrUpdate(_ context.Context, _ string, req insights.DiagnosticSettingsResource, _ string) (insights.DiagnosticSettingsResource, error) {
	cli.rr <- req
	return insights.DiagnosticSettingsResource{}, cli.createUpdErr
}

func (cli *mockDiagnosticSettingsClient) Delete(context.Context, string, string) (autorest.Response, error) {
	return autorest.Response{}, cli.deleteErr
}

var errFake = errors.New("fake error")
