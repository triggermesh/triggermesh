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
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgotesting "k8s.io/client-go/testing"
	"knative.dev/eventing/pkg/reconciler/source"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	rt "knative.dev/pkg/reconciler/testing"

	azureeventgrid "github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/azureeventgridsource"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/eventgrid"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	. "github.com/triggermesh/triggermesh/pkg/sources/reconciler/testing"
	eventtesting "github.com/triggermesh/triggermesh/pkg/sources/testing/event"
)

// adapterCfg is used in every instance of Reconciler defined in reconciler tests.
var adapterCfg = &adapterConfig{
	Image:   "registry/image:tag",
	configs: &source.EmptyVarsGenerator{},
}

func TestReconcileSource(t *testing.T) {
	ctor := reconcilerCtor(adapterCfg)
	src := newEventSource()
	ab := adapterBuilder(adapterCfg)

	TestReconcileAdapter(t, ctor, src, ab)
}

// reconcilerCtor returns a Ctor for a source Reconciler.
func reconcilerCtor(cfg *adapterConfig) Ctor {
	return func(t *testing.T, ctx context.Context, tr *rt.TableRow, ls *Listers) controller.Reconciler {
		stCli := &mockedSystemTopicsClient{
			sysTopics: getMockSystemTopics(tr),
		}

		prCli := &mockedProvidersClient{}

		rgCli := (eventgrid.ResourceGroupsClient)(nil) // unused since tScope isn't an Azure subscription

		esCli := &mockedEventSubscriptionsClient{
			eventSubs: getMockEventSubscriptions(tr),
		}

		ehCli := &mockedEventHubsClient{}

		// inject clients into test data so that table tests can perform
		// assertions on them
		if tr.OtherTestData == nil {
			tr.OtherTestData = make(map[string]interface{}, 2)
		}
		tr.OtherTestData[testSystemTopicsClientDataKey] = stCli
		tr.OtherTestData[testEventSubscriptionsClientDataKey] = esCli

		r := &Reconciler{
			cg:         staticClientGetter(stCli, prCli, rgCli, esCli, ehCli),
			base:       NewTestDeploymentReconciler(ctx, ls),
			adapterCfg: cfg,
			srcLister:  ls.GetAzureEventGridSourceLister().AzureEventGridSources,
		}

		return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
			fakeinjectionclient.Get(ctx), ls.GetAzureEventGridSourceLister(),
			controller.GetEventRecorder(ctx), r)
	}
}

// newEventSource returns a test source object with a minimal set of pre-filled attributes.
func newEventSource() *v1alpha1.AzureEventGridSource {
	src := &v1alpha1.AzureEventGridSource{
		Spec: v1alpha1.AzureEventGridSourceSpec{
			Scope: tScope,
			EventTypes: []string{
				"Microsoft.Storage.BlobCreated",
				"Microsoft.Storage.BlobDeleted",
			},
			Endpoint: v1alpha1.AzureEventGridSourceEndpoint{
				EventHubs: v1alpha1.AzureEventGridSourceDestinationEventHubs{
					NamespaceID: tEventHubNamespaceID,
					HubName:     &tEventHubID.ResourceName,
				},
			},
			Auth: v1alpha1.AzureAuth{
				ServicePrincipal: &v1alpha1.AzureServicePrincipal{
					TenantID: v1alpha1.ValueFromField{
						Value: "00000000-0000-0000-0000-000000000000",
					},
					ClientID: v1alpha1.ValueFromField{
						Value: "00000000-0000-0000-0000-000000000000",
					},
					ClientSecret: v1alpha1.ValueFromField{
						Value: "some_secret",
					},
				},
			},
		},
	}

	// assume finalizer is already set to prevent the generated reconciler
	// from generating an extra Patch action
	src.Finalizers = []string{sources.AzureEventGridSourceResource.String()}

	Populate(src)

	return src
}

// adapterBuilder returns a slim Reconciler containing only the fields accessed
// by r.BuildAdapter().
func adapterBuilder(cfg *adapterConfig) common.AdapterDeploymentBuilder {
	return &Reconciler{
		adapterCfg: cfg,
	}
}

// TestReconcileSubscription contains tests specific to the Azure Event Grid source.
func TestReconcileSubscription(t *testing.T) {
	testCases := rt.TableTest{
		// Regular lifecycle

		{
			Name: "Not yet subscribed",
			Key:  tKey,
			Objects: []runtime.Object{
				newReconciledSource(),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
				Object: newReconciledSource(subscribed),
			}},
			WantEvents: []string{
				createdSystemTopicEvent(),
				createdEventSubsEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledListSystemTopics(true),
				calledCreateUpdateSystemTopic(true),
				calledGetEventSubscription(true),
				calledCreateUpdateEventSubscription(true),
			},
		},
		{
			Name:          "Not yet subscribed, system topic exists",
			Key:           tKey,
			OtherTestData: makeMockSystemTopics(true),
			Objects: []runtime.Object{
				newReconciledSource(subscribed),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			WantEvents: []string{
				createdEventSubsEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledListSystemTopics(true),
				calledCreateUpdateSystemTopic(false),
				calledGetEventSubscription(true),
				calledCreateUpdateEventSubscription(true),
			},
		},
		{
			Name: "Already subscribed and up-to-date",
			Key:  tKey,
			OtherTestData: mergeTableRowData(
				makeMockSystemTopics(true),
				makeMockEventSubscriptions(),
			),
			Objects: []runtime.Object{
				newReconciledSource(subscribed),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledListSystemTopics(true),
				calledCreateUpdateSystemTopic(false),
				calledGetEventSubscription(true),
				calledCreateUpdateEventSubscription(false),
			},
		},
		{
			Name: "Already subscribed but outdated",
			Key:  tKey,
			OtherTestData: mergeTableRowData(
				makeMockSystemTopics(true),
				makeMockEventSubscriptions(outOfSync),
			),
			Objects: []runtime.Object{
				newReconciledSource(subscribed),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			WantEvents: []string{
				updatedEventSubsEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledListSystemTopics(true),
				calledCreateUpdateSystemTopic(false),
				calledGetEventSubscription(true),
				calledCreateUpdateEventSubscription(true),
			},
		},
		{
			Name: "System topic is orphan",
			Key:  tKey,
			OtherTestData: mergeTableRowData(
				makeMockSystemTopics(false),
				makeMockEventSubscriptions(),
			),
			Objects: []runtime.Object{
				newReconciledSource(subscribed),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			WantEvents: []string{
				reOwnedSystemTopicEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledListSystemTopics(true),
				calledCreateUpdateSystemTopic(true),
				calledGetEventSubscription(true),
				calledCreateUpdateEventSubscription(false),
			},
		},

		// Finalization

		{
			Name: "Deletion while subscribed",
			Key:  tKey,
			OtherTestData: mergeTableRowData(
				makeMockSystemTopics(true),
				makeMockEventSubscriptions(),
			),
			Objects: []runtime.Object{
				newReconciledSource(subscribed, deleted),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			WantPatches: []clientgotesting.PatchActionImpl{
				unsetFinalizerPatch(),
			},
			WantEvents: []string{
				deletedEventSubsEvent(),
				deletedSystemTopicEvent(),
				finalizedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledListSystemTopics(true),
				calledListEventSubscriptionsByTopic(true),
				calledDeleteEventSubscription(true),
				calledCreateUpdateSystemTopic(false),
				calledDeleteSystemTopic(true),
			},
		},
		{
			Name:          "Deletion while not subscribed",
			Key:           tKey,
			OtherTestData: makeMockSystemTopics(true),
			Objects: []runtime.Object{
				newReconciledSource(deleted),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			WantPatches: []clientgotesting.PatchActionImpl{
				unsetFinalizerPatch(),
			},
			WantEvents: []string{
				skippedDeleteEventSubsEvent(),
				deletedSystemTopicEvent(),
				finalizedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledListSystemTopics(true),
				calledListEventSubscriptionsByTopic(true),
				calledDeleteEventSubscription(true),
				calledCreateUpdateSystemTopic(false),
				calledDeleteSystemTopic(true),
			},
		},
		{
			Name: "Deletion while topic already gone",
			Key:  tKey,
			Objects: []runtime.Object{
				newReconciledSource(deleted),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			WantPatches: []clientgotesting.PatchActionImpl{
				unsetFinalizerPatch(),
			},
			WantEvents: []string{
				skippedDeleteEventSubsNoTopicEvent(),
				skippedDeleteSystemTopicEvent(),
				finalizedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledListSystemTopics(true),
				calledListEventSubscriptionsByTopic(false),
				calledDeleteEventSubscription(false),
				calledCreateUpdateSystemTopic(false),
				calledDeleteSystemTopic(false),
			},
		},
		{
			Name: "Deletion while topic has remaining subscriptions",
			Key:  tKey,
			OtherTestData: mergeTableRowData(
				makeMockSystemTopics(true),
				makeMockEventSubscriptions(additionalSub),
			),
			Objects: []runtime.Object{
				newReconciledSource(deleted),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			WantPatches: []clientgotesting.PatchActionImpl{
				unsetFinalizerPatch(),
			},
			WantEvents: []string{
				deletedEventSubsEvent(),
				skippedDeleteSystemTopicHasSubsEvent(),
				orphanedSystemTopicEvent(),
				finalizedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledListSystemTopics(true),
				calledListEventSubscriptionsByTopic(true),
				calledDeleteEventSubscription(true),
				calledCreateUpdateSystemTopic(true),
				calledDeleteSystemTopic(false),
			},
		},
		{
			Name: "Deletion while topic has remaining subscriptions and a different owner",
			Key:  tKey,
			OtherTestData: mergeTableRowData(
				makeMockSystemTopics(false),
				makeMockEventSubscriptions(additionalSub),
			),
			Objects: []runtime.Object{
				newReconciledSource(deleted),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			WantPatches: []clientgotesting.PatchActionImpl{
				unsetFinalizerPatch(),
			},
			WantEvents: []string{
				deletedEventSubsEvent(),
				skippedDeleteSystemTopicHasSubsEvent(),
				finalizedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledListSystemTopics(true),
				calledListEventSubscriptionsByTopic(true),
				calledDeleteEventSubscription(true),
				calledCreateUpdateSystemTopic(false),
				calledDeleteSystemTopic(false),
			},
		},
	}

	ctor := reconcilerCtor(adapterCfg)

	testCases.Test(t, MakeFactory(ctor))
}

// tNs/tName match the namespace/name set by (reconciler/testing).Populate.
const (
	tNs   = "testns"
	tName = "test"
	tKey  = tNs + "/" + tName
)

var (
	tSinkURI = &apis.URL{
		Scheme: "http",
		Host:   "default.default.svc.example.com",
		Path:   "/",
	}

	tScope = v1alpha1.AzureResourceID{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		ResourceGroup:    "MyGroup",
		ResourceProvider: "Microsoft.Storage",
		ResourceType:     "storageAccounts",
		ResourceName:     "mystorageaccount",
	}

	tSystemTopicID = v1alpha1.AzureResourceID{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		ResourceGroup:    "MyGroup",
		ResourceProvider: "Microsoft.EventGrid",
		ResourceType:     "systemTopics",
		ResourceName: "io-triggermesh-azureeventgridsources-" +
			"1896707677", // CRC-32 checksum of "<tScope>"
	}

	tEventHubNamespaceID = v1alpha1.AzureResourceID{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		ResourceGroup:    "MyGroup",
		ResourceProvider: "Microsoft.EventHub",
		ResourceType:     "namespaces",
		ResourceName:     "MyNamespace",
	}

	tEventHubID = v1alpha1.AzureResourceID{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		ResourceGroup:    "MyGroup",
		ResourceProvider: "Microsoft.EventHub",
		Namespace:        "MyNamespace",
		ResourceType:     "eventhubs",
		ResourceName:     "MyEventHub",
	}

	tEventSubscriptionID = v1alpha1.AzureResourceID{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		ResourceGroup:    "MyGroup",
		ResourceProvider: "Microsoft.EventGrid",
		ResourceType:     "systemTopics",
		ResourceName: "io-triggermesh-azureeventgridsources-" +
			"1896707677", // CRC-32 checksum of "<tScope>"
		SubResourceType: "eventSubscriptions",
		SubResourceName: "io-triggermesh-azureeventgridsources-" +
			"521367233", // CRC-32 checksum of "<tNs>/<tName>"
	}
)

/* Source and receive adapter */

// sourceOption is a functional option for an event source.
type sourceOption func(*v1alpha1.AzureEventGridSource)

// newReconciledSource returns a test event source object that is identical to
// what ReconcileKind generates.
func newReconciledSource(opts ...sourceOption) *v1alpha1.AzureEventGridSource {
	src := newEventSource()

	// assume the sink URI is resolved
	src.Spec.Sink.Ref = nil
	src.Spec.Sink.URI = tSinkURI

	// assume status conditions are already set to True to ensure
	// ReconcileKind is a no-op
	status := src.GetStatusManager()
	status.MarkSink(tSinkURI)
	status.PropagateDeploymentAvailability(context.Background(), newReconciledAdapter(), nil)

	for _, opt := range opts {
		opt(src)
	}

	return src
}

// subscribed sets the Subscribed status condition to True and reports the
// resource IDs of the Event Grid subscription and the destination Event Hub in
// the source's status.
func subscribed(src *v1alpha1.AzureEventGridSource) {
	src.Status.MarkSubscribed()
	src.Status.EventSubscriptionID = &tEventSubscriptionID
	src.Status.EventHubID = &tEventHubID
}

// deleted marks the source as deleted.
func deleted(src *v1alpha1.AzureEventGridSource) {
	t := metav1.Unix(0, 0)
	src.SetDeletionTimestamp(&t)
}

// newReconciledServiceAccount returns a test ServiceAccount object that is
// identical to what ReconcileKind generates.
func newReconciledServiceAccount() *corev1.ServiceAccount {
	return NewServiceAccount(newEventSource())()
}

// newReconciledRoleBinding returns a test RoleBinding object that is
// identical to what ReconcileKind generates.
func newReconciledRoleBinding() *rbacv1.RoleBinding {
	return NewRoleBinding(newReconciledServiceAccount())()
}

// newReconciledAdapter returns a test receive adapter object that is identical
// to what ReconcileKind generates.
func newReconciledAdapter() *appsv1.Deployment {
	// hack: we need to pass a source which has status.eventHubID already
	// set for the deployment to contain an AZURE_HUB_NAME env var with the
	// expected value
	src := newEventSource()
	src.Status.EventHubID = &tEventHubID

	adapter := adapterBuilder(adapterCfg).BuildAdapter(src, tSinkURI)

	adapter.Status.Conditions = []appsv1.DeploymentCondition{{
		Type:   appsv1.DeploymentAvailable,
		Status: corev1.ConditionTrue,
	}}

	return adapter
}

// mergeTableRowData flattens multiple maps of TableRow data.
func mergeTableRowData(data ...map[string]interface{}) map[string]interface{} {
	trData := make(map[string]interface{})

	for _, d := range data {
		for k, v := range d {
			trData[k] = v
		}
	}

	return trData
}

/* Azure clients */

// staticClientGetter transforms the given client interfaces into a
// ClientGetter.
func staticClientGetter(stCli eventgrid.SystemTopicsClient, prCli eventgrid.ProvidersClient,
	rgCli eventgrid.ResourceGroupsClient, esCli eventgrid.EventSubscriptionsClient,
	ehCli eventgrid.EventHubsClient) eventgrid.ClientGetterFunc {

	return func(*v1alpha1.AzureEventGridSource) (
		eventgrid.SystemTopicsClient,
		eventgrid.ProvidersClient,
		eventgrid.ResourceGroupsClient,
		eventgrid.EventSubscriptionsClient,
		eventgrid.EventHubsClient,
		error) {

		return stCli, prCli, rgCli, esCli, ehCli, nil
	}
}

type mockedSystemTopicsClient struct {
	eventgrid.SystemTopicsClient

	sysTopics mockSystemTopics

	calledList         bool
	calledCreateUpdate bool
	calledDelete       bool
}

// the fake client expects keys in the format <resource group>#<topic name>
type mockSystemTopics map[string]azureeventgrid.SystemTopic

const testSystemTopicsClientDataKey = "stClient"

func (c *mockedSystemTopicsClient) ListBySubscriptionComplete(ctx context.Context,
	filter string, top *int32) (azureeventgrid.SystemTopicsListResultIterator, error) {

	c.calledList = true

	// Page 1 contains only dummy values to force the iteration to page 2.
	// This is to ensure that our code uses the page iterator properly.
	sysTopicsPage1 := []azureeventgrid.SystemTopic{
		newMockSystemTopic("/dummy1"),
		newMockSystemTopic("/dummy2"),
	}

	// Page 2 contains the real mock SystemTopics.
	var sysTopicsPage2 []azureeventgrid.SystemTopic
	for _, st := range c.sysTopics {
		sysTopicsPage2 = append(sysTopicsPage2, st)
	}

	results := azureeventgrid.SystemTopicsListResult{
		Value:    &sysTopicsPage1,
		NextLink: to.StringPtr("page2"),
	}

	getNextPage := func(_ context.Context, prev azureeventgrid.SystemTopicsListResult) (azureeventgrid.SystemTopicsListResult, error) {
		if next := prev.NextLink; next != nil && *next == "page2" {
			return azureeventgrid.SystemTopicsListResult{
				Value: &sysTopicsPage2,
			}, nil
		}
		// returning an empty list (empty Value field) stops the iteration
		return azureeventgrid.SystemTopicsListResult{}, nil
	}

	page := azureeventgrid.NewSystemTopicsListResultPage(results, getNextPage)

	return azureeventgrid.NewSystemTopicsListResultIterator(page), nil
}

func (c *mockedSystemTopicsClient) CreateOrUpdate(ctx context.Context,
	resourceGroupName, systemTopicName string, systemTopicInfo azureeventgrid.SystemTopic) (
	azureeventgrid.SystemTopicsCreateOrUpdateFuture, error) {

	c.calledCreateUpdate = true

	return azureeventgrid.SystemTopicsCreateOrUpdateFuture{
		FutureAPI: (*mockedFuture)(nil),
		Result: func(azureeventgrid.SystemTopicsClient) (azureeventgrid.SystemTopic, error) {
			st := systemTopicInfo
			st.ID = to.StringPtr(tSystemTopicID.String())
			return st, nil
		},
	}, nil
}

func (c *mockedSystemTopicsClient) Delete(ctx context.Context,
	resourceGroupName, systemTopicName string) (azureeventgrid.SystemTopicsDeleteFuture, error) {

	c.calledDelete = true

	if len(c.sysTopics) == 0 {
		return azureeventgrid.SystemTopicsDeleteFuture{}, notFoundAzureErr()
	}

	if _, ok := c.sysTopics[resourceGroupName+"#"+systemTopicName]; !ok {
		return azureeventgrid.SystemTopicsDeleteFuture{}, notFoundAzureErr()
	}

	delete(c.sysTopics, resourceGroupName+"#"+systemTopicName)

	return azureeventgrid.SystemTopicsDeleteFuture{
		FutureAPI: (*mockedFuture)(nil),
	}, nil
}

func (c *mockedSystemTopicsClient) BaseClient() autorest.Client {
	return autorest.Client{}
}

func (c *mockedSystemTopicsClient) ConcreteClient() azureeventgrid.SystemTopicsClient {
	return azureeventgrid.SystemTopicsClient{}
}

const mockSystemTopicsDataKey = "sysTopics"

// makeMockSystemTopics returns a mocked list of system topics to be used as
// TableRow data.
func makeMockSystemTopics(hasOwner bool) map[string]interface{} {
	st := newMockSystemTopic(tScope.String())
	st.ID = to.StringPtr(tSystemTopicID.String())
	st.Name = &tSystemTopicID.ResourceName

	if hasOwner {
		st.Tags = map[string]*string{
			eventgridTagOwnerResource:  to.StringPtr(sources.AzureEventGridSourceResource.String()),
			eventgridTagOwnerNamespace: to.StringPtr(newEventSource().Namespace),
			eventgridTagOwnerName:      to.StringPtr(newEventSource().Name),
		}
	}

	// key format expected by mocked client impl
	sysTopicKey := tSystemTopicID.ResourceGroup + "#" + tSystemTopicID.ResourceName

	return map[string]interface{}{
		mockSystemTopicsDataKey: mockSystemTopics{
			sysTopicKey: st,
		},
	}
}

// newMockSystemTopic returns a SystemTopic which Source attribute matches the
// given scope.
func newMockSystemTopic(scope string) azureeventgrid.SystemTopic {
	return azureeventgrid.SystemTopic{
		SystemTopicProperties: &azureeventgrid.SystemTopicProperties{
			Source: &scope,
		},
	}
}

// getMockSystemTopics gets mocked system topics from the TableRow's data.
func getMockSystemTopics(tr *rt.TableRow) mockSystemTopics {
	sysTopics, ok := tr.OtherTestData[mockSystemTopicsDataKey]
	if !ok {
		return nil
	}
	return sysTopics.(mockSystemTopics)
}

func calledListSystemTopics(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testSystemTopicsClientDataKey].(*mockedSystemTopicsClient)

		if expectCall && !cli.calledList {
			t.Error("Did not call ListBySubscriptionComplete() on system topics")
		}
		if !expectCall && cli.calledList {
			t.Error("Unexpected call to ListBySubscriptionComplete() on system topics")
		}
	}
}
func calledCreateUpdateSystemTopic(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testSystemTopicsClientDataKey].(*mockedSystemTopicsClient)

		if expectCall && !cli.calledCreateUpdate {
			t.Error("Did not call CreateOrUpdate() on system topic")
		}
		if !expectCall && cli.calledCreateUpdate {
			t.Error("Unexpected call to CreateOrUpdate() on system topic")
		}
	}
}
func calledDeleteSystemTopic(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testSystemTopicsClientDataKey].(*mockedSystemTopicsClient)

		if expectCall && !cli.calledDelete {
			t.Error("Did not call Delete() on system topic")
		}
		if !expectCall && cli.calledDelete {
			t.Error("Unexpected call to Delete() on system topic")
		}
	}
}

type mockedProvidersClient struct {
	eventgrid.ProvidersClient
}

func (c *mockedProvidersClient) Get(ctx context.Context, resourceProviderNamespace, expand string) (resources.Provider, error) {
	// based on the global tScope variable, we assume the provider is
	// always "Microsoft.Storage" in tests
	resourceTypes := []resources.ProviderResourceType{
		{
			ResourceType:      to.StringPtr("storageAccounts"),
			DefaultAPIVersion: to.StringPtr("1970-01-01"),
		},
	}

	return resources.Provider{
		ResourceTypes: &resourceTypes,
	}, nil
}

type mockedEventSubscriptionsClient struct {
	eventgrid.EventSubscriptionsClient

	eventSubs mockEventSubscriptions

	calledGet          bool
	calledList         bool
	calledCreateUpdate bool
	calledDelete       bool
}

// the fake client expects keys in the format <resource group>#<topic name>#<subscription name>
type mockEventSubscriptions map[string]azureeventgrid.EventSubscription

const testEventSubscriptionsClientDataKey = "esClient"

func (c *mockedEventSubscriptionsClient) Get(ctx context.Context, resourceGroupName, systemTopicName,
	eventSubscriptionName string) (azureeventgrid.EventSubscription, error) {

	c.calledGet = true

	if len(c.eventSubs) == 0 {
		return azureeventgrid.EventSubscription{}, notFoundAzureErr()
	}

	sub, ok := c.eventSubs[resourceGroupName+"#"+systemTopicName+"#"+eventSubscriptionName]
	if !ok {
		return azureeventgrid.EventSubscription{}, notFoundAzureErr()
	}

	return sub, nil
}

func (c *mockedEventSubscriptionsClient) ListBySystemTopic(ctx context.Context, resourceGroupName, systemTopicName,
	filter string, top *int32) (azureeventgrid.EventSubscriptionsListResultPage, error) {

	c.calledList = true

	var subsPage []azureeventgrid.EventSubscription

	for _, subs := range c.eventSubs {
		// assume that all mocked event subscriptions belong to the
		// mocked system topic
		subsPage = append(subsPage, subs)
	}

	results := azureeventgrid.EventSubscriptionsListResult{
		Value: &subsPage,
	}

	// getNextPage can be nil because we never iterate over the result of ListBySystemTopic
	page := azureeventgrid.NewEventSubscriptionsListResultPage(results, nil)

	return page, nil
}

func (c *mockedEventSubscriptionsClient) CreateOrUpdate(ctx context.Context, resourceGroupName, systemTopicName,
	eventSubscriptionName string, eventSubscriptionInfo azureeventgrid.EventSubscription) (
	azureeventgrid.SystemTopicEventSubscriptionsCreateOrUpdateFuture, error) {

	c.calledCreateUpdate = true

	return azureeventgrid.SystemTopicEventSubscriptionsCreateOrUpdateFuture{
		FutureAPI: (*mockedFuture)(nil),
		Result: func(azureeventgrid.SystemTopicEventSubscriptionsClient) (azureeventgrid.EventSubscription, error) {
			subs := eventSubscriptionInfo
			subs.ID = to.StringPtr(tEventSubscriptionID.String())
			return subs, nil
		},
	}, nil
}

func (c *mockedEventSubscriptionsClient) Delete(ctx context.Context, resourceGroupName, systemTopicName,
	eventSubscriptionName string) (azureeventgrid.SystemTopicEventSubscriptionsDeleteFuture, error) {

	c.calledDelete = true

	if len(c.eventSubs) == 0 {
		return azureeventgrid.SystemTopicEventSubscriptionsDeleteFuture{}, notFoundAzureErr()
	}

	if _, ok := c.eventSubs[resourceGroupName+"#"+systemTopicName+"#"+eventSubscriptionName]; !ok {
		return azureeventgrid.SystemTopicEventSubscriptionsDeleteFuture{}, notFoundAzureErr()
	}

	delete(c.eventSubs, resourceGroupName+"#"+systemTopicName+"#"+eventSubscriptionName)

	return azureeventgrid.SystemTopicEventSubscriptionsDeleteFuture{
		FutureAPI: (*mockedFuture)(nil),
	}, nil
}

func (c *mockedEventSubscriptionsClient) BaseClient() autorest.Client {
	return autorest.Client{}
}

func (c *mockedEventSubscriptionsClient) ConcreteClient() azureeventgrid.SystemTopicEventSubscriptionsClient {
	return azureeventgrid.SystemTopicEventSubscriptionsClient{}
}

const mockEventSubscriptionsDataKey = "eventSubs"

// makeMockEventSubscriptions returns a mocked list of event subscriptions to
// be used as TableRow data.
func makeMockEventSubscriptions(opts ...mockEventSubscriptionsOption) map[string]interface{} {
	subs := newEventSubscription(tEventHubID.String(), newEventSource().GetEventTypes())
	subs.ID = to.StringPtr(tEventSubscriptionID.String())

	// key format expected by mocked client impl
	subKey := tEventSubscriptionID.ResourceGroup + "#" + tEventSubscriptionID.ResourceName + "#" + tEventSubscriptionID.SubResourceName

	mockEventSubscriptionsData := mockEventSubscriptions{
		subKey: subs,
	}

	for _, opt := range opts {
		opt(mockEventSubscriptionsData)
	}

	return map[string]interface{}{
		mockEventSubscriptionsDataKey: mockEventSubscriptionsData,
	}
}

type mockEventSubscriptionsOption func(mockEventSubscriptions)

// outOfSync is a mockEventSubscriptionsOption that injects an arbitrary change
// to the "main" mocked event subscription to cause the comparison to be false
// in the reconciler.
func outOfSync(ms mockEventSubscriptions) {
	// key format expected by mocked client impl
	subKey := tEventSubscriptionID.ResourceGroup + "#" + tEventSubscriptionID.ResourceName + "#" + tEventSubscriptionID.SubResourceName

	*ms[subKey].RetryPolicy.EventTimeToLiveInMinutes++
}

// additionalSub is a mockEventSubscriptionsOption that injects an additional
// empty EventSubscription to the existing list of mocked subscriptions.
// Used by (*mockedEventSubscriptionsClient).ListBySystemTopic to determine
// whether a system topic has remaining subscriptions.
func additionalSub(ms mockEventSubscriptions) {
	randKey := strconv.FormatInt(rand.NewSource(time.Now().Unix()).Int63(), 10)
	ms[randKey] = azureeventgrid.EventSubscription{}
}

// getMockEventSubscriptions gets mocked event subscriptions from the
// TableRow's data.
func getMockEventSubscriptions(tr *rt.TableRow) mockEventSubscriptions {
	subs, ok := tr.OtherTestData[mockEventSubscriptionsDataKey]
	if !ok {
		return nil
	}
	return subs.(mockEventSubscriptions)
}

func calledGetEventSubscription(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testEventSubscriptionsClientDataKey].(*mockedEventSubscriptionsClient)

		if expectCall && !cli.calledGet {
			t.Error("Did not call Get() on event subscription")
		}
		if !expectCall && cli.calledGet {
			t.Error("Unexpected call to Get() on event subscription")
		}
	}
}
func calledListEventSubscriptionsByTopic(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testEventSubscriptionsClientDataKey].(*mockedEventSubscriptionsClient)

		if expectCall && !cli.calledList {
			t.Error("Did not call ListBySystemTopic() on event subscriptions")
		}
		if !expectCall && cli.calledList {
			t.Error("Unexpected call to ListBySystemTopic() on event subscriptions")
		}
	}
}
func calledCreateUpdateEventSubscription(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testEventSubscriptionsClientDataKey].(*mockedEventSubscriptionsClient)

		if expectCall && !cli.calledCreateUpdate {
			t.Error("Did not call CreateOrUpdate() on event subscription")
		}
		if !expectCall && cli.calledCreateUpdate {
			t.Error("Unexpected call to CreateOrUpdate() on event subscription")
		}
	}
}
func calledDeleteEventSubscription(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testEventSubscriptionsClientDataKey].(*mockedEventSubscriptionsClient)

		if expectCall && !cli.calledDelete {
			t.Error("Did not call Delete() on event subscription")
		}
		if !expectCall && cli.calledDelete {
			t.Error("Unexpected call to Delete() on event subscription")
		}
	}
}

type mockedEventHubsClient struct {
	eventgrid.EventHubsClient
}

func (c *mockedEventHubsClient) Get(ctx context.Context, rg, ns, name string) (eventhub.Model, error) {
	return eventhub.Model{}, nil
}

func (c *mockedEventHubsClient) CreateOrUpdate(ctx context.Context, rg, ns, name string, params eventhub.Model) (eventhub.Model, error) {
	return eventhub.Model{}, nil
}

func (c *mockedEventHubsClient) Delete(ctx context.Context, rg, ns, name string) (autorest.Response, error) {
	return autorest.Response{}, nil
}

func notFoundAzureErr() error {
	return autorest.DetailedError{
		StatusCode: http.StatusNotFound,
	}
}

type mockedFuture struct {
	azure.FutureAPI
}

func (*mockedFuture) WaitForCompletionRef(context.Context, autorest.Client) error {
	return nil
}

/* Patches */

func unsetFinalizerPatch() clientgotesting.PatchActionImpl {
	return clientgotesting.PatchActionImpl{
		Name:      tName,
		PatchType: types.MergePatchType,
		Patch:     []byte(`{"metadata":{"finalizers":[],"resourceVersion":""}}`),
	}
}

/* Events */

func createdSystemTopicEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonSystemTopicSynced,
		"Created system topic %q for resource %q", &tSystemTopicID, &tScope)
}
func reOwnedSystemTopicEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonSystemTopicSynced,
		"Re-owned orphan system topic %q", &tSystemTopicID)
}
func deletedSystemTopicEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonSystemTopicFinalized,
		"Deleted system topic %q", &tSystemTopicID)
}
func orphanedSystemTopicEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonSystemTopicFinalized,
		"Removed ownership tags on system topic %q", &tSystemTopicID)
}
func skippedDeleteSystemTopicEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, ReasonSystemTopicFinalized,
		"System topic not found, skipping finalization")
}
func skippedDeleteSystemTopicHasSubsEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, ReasonSystemTopicFinalized,
		"System topic has remaining event subscriptions, skipping deletion")
}
func createdEventSubsEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonSubscribed,
		"Created event subscription %q", &tEventSubscriptionID)
}
func updatedEventSubsEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonSubscribed,
		"Updated event subscription %q", &tEventSubscriptionID)
}
func deletedEventSubsEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonUnsubscribed,
		"Deleted event subscription %q from system topic %q", tEventSubscriptionID.SubResourceName, &tSystemTopicID)
}
func skippedDeleteEventSubsEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, ReasonUnsubscribed,
		"Event subscription %q not found, skipping deletion", tEventSubscriptionID.SubResourceName)
}
func skippedDeleteEventSubsNoTopicEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, ReasonUnsubscribed,
		"System topic not found, skipping finalization of event subscription")
}
func finalizedEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, "FinalizerUpdate", "Updated %q finalizers", tName)
}
