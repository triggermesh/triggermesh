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

package azureblobstoragesource

import (
	"context"
	"net/http"
	"testing"

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

	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/mgmt/eventgrid"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventhub/mgmt/eventhub"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/azureblobstoragesource"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/storage"
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
		esCli := &mockedEventSubscriptionsClient{
			eventSubs: getMockEventSubscriptions(tr),
		}

		ehCli := &mockedEventHubsClient{}

		// inject clients into test data so that table tests can perform
		// assertions on it
		if tr.OtherTestData == nil {
			tr.OtherTestData = make(map[string]interface{}, 2)
		}
		tr.OtherTestData[testEventSubscriptionsClientDataKey] = esCli
		tr.OtherTestData[testEventHubsClientDataKey] = ehCli

		r := &Reconciler{
			cg:         staticClientGetter(esCli, ehCli),
			base:       NewTestDeploymentReconciler(ctx, ls),
			adapterCfg: cfg,
			srcLister:  ls.GetAzureBlobStorageSourceLister().AzureBlobStorageSources,
		}

		return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
			fakeinjectionclient.Get(ctx), ls.GetAzureBlobStorageSourceLister(),
			controller.GetEventRecorder(ctx), r)
	}
}

// newEventSource returns a test source object with a minimal set of pre-filled attributes.
func newEventSource() *v1alpha1.AzureBlobStorageSource {
	src := &v1alpha1.AzureBlobStorageSource{
		Spec: v1alpha1.AzureBlobStorageSourceSpec{
			StorageAccountID: tStorageAccID,
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
	src.Finalizers = []string{sources.AzureBlobStorageSourceResource.String()}

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

// TestReconcileSubscription contains tests specific to the Azure Blob Storage source.
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
				createdEventSubsEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledGetEventSubscription(true),
				calledCreateUpdateEventSubscription(true),
			},
		},
		{
			Name:          "Already subscribed and up-to-date",
			Key:           tKey,
			OtherTestData: makeMockEventSubscriptions(true),
			Objects: []runtime.Object{
				newReconciledSource(subscribed),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledGetEventSubscription(true),
				calledCreateUpdateEventSubscription(false),
			},
		},
		{
			Name:          "Already subscribed but outdated",
			Key:           tKey,
			OtherTestData: makeMockEventSubscriptions(false),
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
				calledGetEventSubscription(true),
				calledCreateUpdateEventSubscription(true),
			},
		},

		// Finalization

		{
			Name:          "Deletion while subscribed",
			Key:           tKey,
			OtherTestData: makeMockEventSubscriptions(true),
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
				finalizedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledGetEventSubscription(false),
				calledCreateUpdateEventSubscription(false),
				calledDeleteEventSubscription(true),
			},
		},
		{
			Name: "Deletion while not subscribed",
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
				skippedDeleteEventSubsEvent(),
				finalizedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledGetEventSubscription(false),
				calledCreateUpdateEventSubscription(false),
				calledDeleteEventSubscription(true),
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

	tEventSubs = "io.triggermesh.azureblobstoragesource." + tNs + "." + tName
)

var (
	tSinkURI = &apis.URL{
		Scheme: "http",
		Host:   "default.default.svc.example.com",
		Path:   "/",
	}

	tStorageAccID = v1alpha1.StorageAccountResourceID{
		SubscriptionID: "00000000-0000-0000-0000-000000000000",
		ResourceGroup:  "MyGroup",
		StorageAccount: "mystorageaccount",
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
)

/* Source and receive adapter */

// sourceOption is a functional option for an event source.
type sourceOption func(*v1alpha1.AzureBlobStorageSource)

// newReconciledSource returns a test event source object that is identical to
// what ReconcileKind generates.
func newReconciledSource(opts ...sourceOption) *v1alpha1.AzureBlobStorageSource {
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
// resource ID of the destination Event Hub in the source's status.
func subscribed(src *v1alpha1.AzureBlobStorageSource) {
	src.Status.MarkSubscribed()
	src.Status.EventHubID = &tEventHubID
}

// deleted marks the source as deleted.
func deleted(src *v1alpha1.AzureBlobStorageSource) {
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

/* Azure clients */

// staticClientGetter transforms the given client interfaces into a
// ClientGetter.
func staticClientGetter(esCli storage.EventSubscriptionsClient, ehCli storage.EventHubsClient) storage.ClientGetterFunc {
	return func(*v1alpha1.AzureBlobStorageSource) (storage.EventSubscriptionsClient, storage.EventHubsClient, error) {
		return esCli, ehCli, nil
	}
}

type mockedEventSubscriptionsClient struct {
	storage.EventSubscriptionsClient

	eventSubs mockEventSubscriptions

	calledGet          bool
	calledCreateUpdate bool
	calledDelete       bool
}

// the fake client expects keys in the format <storage acc id>/<subscription name>
type mockEventSubscriptions map[string]eventgrid.EventSubscription

const testEventSubscriptionsClientDataKey = "esClient"

func (c *mockedEventSubscriptionsClient) Get(ctx context.Context, scope, name string) (eventgrid.EventSubscription, error) {
	c.calledGet = true

	if len(c.eventSubs) == 0 {
		return eventgrid.EventSubscription{}, notFoundAzureErr()
	}

	sub, ok := c.eventSubs[scope+"/"+name]
	if !ok {
		return eventgrid.EventSubscription{}, notFoundAzureErr()
	}

	return sub, nil
}

func (c *mockedEventSubscriptionsClient) CreateOrUpdate(ctx context.Context, scope, name string,
	info eventgrid.EventSubscription) (eventgrid.EventSubscriptionsCreateOrUpdateFuture, error) {

	c.calledCreateUpdate = true
	return eventgrid.EventSubscriptionsCreateOrUpdateFuture{}, nil
}

func (c *mockedEventSubscriptionsClient) Delete(ctx context.Context, scope, name string) (eventgrid.EventSubscriptionsDeleteFuture, error) {
	c.calledDelete = true

	if len(c.eventSubs) == 0 {
		return eventgrid.EventSubscriptionsDeleteFuture{}, notFoundAzureErr()
	}

	var err error
	if _, ok := c.eventSubs[scope+"/"+name]; !ok {
		err = notFoundAzureErr()
	}

	return eventgrid.EventSubscriptionsDeleteFuture{}, err
}

const mockEventSubscriptionsDataKey = "eventSubs"

// makeMockEventSubscriptions returns a mocked list of event subscriptions to
// be used as TableRow data.
func makeMockEventSubscriptions(inSync bool) map[string]interface{} {
	sub := newEventSubscription(tEventHubID.String(), newEventSource().GetEventTypes())
	sub.ID = to.StringPtr("/irrelevant/resource/id")

	if !inSync {
		// inject arbitrary change to cause comparison to be false
		*sub.EventSubscriptionProperties.RetryPolicy.EventTimeToLiveInMinutes++
	}

	// key format expected by mocked client impl
	subKey := tStorageAccID.String() + "/" + tEventSubs

	return map[string]interface{}{
		mockEventSubscriptionsDataKey: mockEventSubscriptions{
			subKey: sub,
		},
	}
}

// getMockEventSubscriptions gets mocked event subscriptions from the
// TableRow's data.
func getMockEventSubscriptions(tr *rt.TableRow) mockEventSubscriptions {
	hubs, ok := tr.OtherTestData[mockEventSubscriptionsDataKey]
	if !ok {
		return nil
	}
	return hubs.(mockEventSubscriptions)
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
	storage.EventHubsClient
}

const testEventHubsClientDataKey = "ehClient"

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

/* Patches */

func unsetFinalizerPatch() clientgotesting.PatchActionImpl {
	return clientgotesting.PatchActionImpl{
		Name:      tName,
		PatchType: types.MergePatchType,
		Patch:     []byte(`{"metadata":{"finalizers":[],"resourceVersion":""}}`),
	}
}

/* Events */

func createdEventSubsEvent() string {
	tStorageAccount := tStorageAccID.StorageAccount
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonSubscribed,
		"Created event subscription %q for storage account %q", tEventSubs, tStorageAccount)
}
func updatedEventSubsEvent() string {
	tStorageAccount := tStorageAccID.StorageAccount
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonSubscribed,
		"Updated event subscription %q for storage account %q", tEventSubs, tStorageAccount)
}
func deletedEventSubsEvent() string {
	tStorageAccount := tStorageAccID.StorageAccount
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonUnsubscribed,
		"Deleted event subscription %q for storage account %q", tEventSubs, tStorageAccount)
}
func skippedDeleteEventSubsEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, ReasonUnsubscribed,
		"Event subscription not found, skipping deletion")
}
func finalizedEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, "FinalizerUpdate", "Updated %q finalizers", tName)
}
