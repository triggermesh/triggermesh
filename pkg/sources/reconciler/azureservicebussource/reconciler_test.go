/*
Copyright 2023 TriggerMesh Inc.

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

package azureservicebussource

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

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

	azservicebus "github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/go-autorest/autorest"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/azureservicebussource"
	common "github.com/triggermesh/triggermesh/pkg/reconciler"
	. "github.com/triggermesh/triggermesh/pkg/reconciler/testing"
	"github.com/triggermesh/triggermesh/pkg/sources/client/azure/servicebus"
	eventtesting "github.com/triggermesh/triggermesh/pkg/testing/event"
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
		subsCli := &mockedSubscriptionsClient{
			subs: getMockSubscriptions(tr),
		}

		// inject clients into test data so that table tests can perform
		// assertions on it
		if tr.OtherTestData == nil {
			tr.OtherTestData = make(map[string]interface{}, 2)
		}
		tr.OtherTestData[testSubscriptionsClientDataKey] = subsCli

		r := &Reconciler{
			cg:         staticClientGetter(subsCli),
			adapterCfg: cfg,
		}

		r.base = NewTestDeploymentReconciler[*v1alpha1.AzureServiceBusSource](ctx, ls,
			ls.GetAzureServiceBusSourceLister().AzureServiceBusSources,
		)

		return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
			fakeinjectionclient.Get(ctx), ls.GetAzureServiceBusSourceLister(),
			controller.GetEventRecorder(ctx), r)
	}
}

// newEventSource returns a test source object with a minimal set of pre-filled attributes.
func newEventSource() *v1alpha1.AzureServiceBusSource {
	src := &v1alpha1.AzureServiceBusSource{
		Spec: v1alpha1.AzureServiceBusSourceSpec{
			TopicID: &tTopicID,
			Auth: v1alpha1.AzureAuth{
				ServicePrincipal: &v1alpha1.AzureServicePrincipal{
					TenantID: commonv1alpha1.ValueFromField{
						Value: "00000000-0000-0000-0000-000000000000",
					},
					ClientID: commonv1alpha1.ValueFromField{
						Value: "00000000-0000-0000-0000-000000000000",
					},
					ClientSecret: commonv1alpha1.ValueFromField{
						Value: "some_secret",
					},
				},
			},
		},
	}

	// assume finalizer is already set to prevent the generated reconciler
	// from generating an extra Patch action
	src.Finalizers = []string{sources.AzureServiceBusSourceResource.String()}

	Populate(src)

	return src
}

// adapterBuilder returns a slim Reconciler containing only the fields accessed
// by r.BuildAdapter().
func adapterBuilder(cfg *adapterConfig) common.AdapterBuilder[*appsv1.Deployment] {
	return &Reconciler{
		adapterCfg: cfg,
	}
}

// TestReconcileSubscription contains tests specific to the Azure Event Grid source.
func TestReconcileSubscription(t *testing.T) {
	newReconciledAdapter := mustNewReconciledAdapter(t)
	newReconciledSource := mustNewReconciledSource(t)

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
				createdSubsEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledGetSubscription(true),
				calledCreateUpdateSubscription(true),
			},
		},
		{
			Name:          "Already subscribed",
			Key:           tKey,
			OtherTestData: makeMockSubscriptions(),
			Objects: []runtime.Object{
				newReconciledSource(subscribed),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledGetSubscription(true),
				calledCreateUpdateSubscription(false),
			},
		},

		// Finalization

		{
			Name:          "Deletion while subscribed",
			Key:           tKey,
			OtherTestData: makeMockSubscriptions(),
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
				deletedSubsEvent(),
				finalizedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledGetSubscription(false),
				calledCreateUpdateSubscription(false),
				calledDeleteSubscription(true),
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
				skippedDeleteSubsEvent(),
				finalizedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledGetSubscription(false),
				calledCreateUpdateSubscription(false),
				calledDeleteSubscription(true),
			},
		},
	}

	ctor := reconcilerCtor(adapterCfg)

	testCases.Test(t, MakeFactory(ctor))
}

// tNs/tName match the namespace/name set by (reconciler/testing).Populate.
const (
	tNs       = "testns"
	tName     = "test"
	tKey      = tNs + "/" + tName
	tCRC32Key = "521367233"
)

var (
	tSinkURI = &apis.URL{
		Scheme: "http",
		Host:   "default.default.svc.example.com",
		Path:   "/",
	}

	tTopicID = v1alpha1.AzureResourceID{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		ResourceGroup:    "MyGroup",
		ResourceProvider: "Microsoft.ServiceBus",
		Namespace:        "MyNamespace",
		ResourceType:     "topics",
		ResourceName:     "MyTopic",
	}

	tSubscriptionID = v1alpha1.AzureResourceID{
		SubscriptionID:   "00000000-0000-0000-0000-000000000000",
		ResourceGroup:    "MyGroup",
		ResourceProvider: "Microsoft.ServiceBus",
		Namespace:        "MyNamespace",
		ResourceType:     "topics",
		ResourceName:     "MyTopic",
		SubResourceType:  "subscriptions",
		SubResourceName:  "io.triggermesh.azureservicebussources-" + tCRC32Key,
	}
)

/* Source and receive adapter */

// sourceOption is a functional option for an event source.
type sourceOption func(*v1alpha1.AzureServiceBusSource)

// newReconciledSource returns a test event source object that is identical to
// what ReconcileKind generates.
func newReconciledSource(opts ...sourceOption) (*v1alpha1.AzureServiceBusSource, error) {
	src := newEventSource()

	// assume the sink URI is resolved
	src.Spec.Sink.Ref = nil
	src.Spec.Sink.URI = tSinkURI

	a, err := newReconciledAdapter()
	if err != nil {
		return nil, err
	}

	// assume status conditions are already set to True to ensure
	// ReconcileKind is a no-op
	status := src.GetStatusManager()
	status.MarkSink(tSinkURI)
	status.PropagateDeploymentAvailability(context.Background(), a, nil)

	for _, opt := range opts {
		opt(src)
	}

	return src, nil
}

func mustNewReconciledSource(t *testing.T) func(...sourceOption) *v1alpha1.AzureServiceBusSource {
	return func(opts ...sourceOption) *v1alpha1.AzureServiceBusSource {
		src, err := newReconciledSource(opts...)
		require.NoError(t, err)
		return src
	}
}

// subscribed sets the Subscribed status condition to True and reports the
// resource ID of the Service Bus Subscription in the source's status.
func subscribed(src *v1alpha1.AzureServiceBusSource) {
	src.Status.MarkSubscribed()
	src.Status.SubscriptionID = &tSubscriptionID
}

// deleted marks the source as deleted.
func deleted(src *v1alpha1.AzureServiceBusSource) {
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
	return NewConfigWatchRoleBinding(newReconciledServiceAccount())()
}

// newReconciledAdapter returns a test receive adapter object that is identical
// to what ReconcileKind generates.
func newReconciledAdapter() (*appsv1.Deployment, error) {
	// hack: we need to pass a source which has status.subscriptionID
	// already set for the deployment to contain a
	// SERVICEBUS_SUBSCRIPTION_RESOURCE_ID env var with the expected value
	src := newEventSource()
	src.Status.SubscriptionID = &tSubscriptionID

	adapter, err := adapterBuilder(adapterCfg).BuildAdapter(src, tSinkURI)
	if err != nil {
		return nil, fmt.Errorf("building adapter object using provided Reconcilable: %w", err)
	}

	adapter.Status.Conditions = []appsv1.DeploymentCondition{{
		Type:   appsv1.DeploymentAvailable,
		Status: corev1.ConditionTrue,
	}}

	return adapter, nil
}

func mustNewReconciledAdapter(t *testing.T) func() *appsv1.Deployment {
	return func() *appsv1.Deployment {
		a, err := newReconciledAdapter()
		require.NoError(t, err)
		return a
	}
}

/* Azure clients */

// staticClientGetter transforms the given client interfaces into a ClientGetter.
func staticClientGetter(esCli servicebus.SubscriptionsClient) servicebus.ClientGetterFunc {
	return func(*v1alpha1.AzureServiceBusSource) (servicebus.SubscriptionsClient, error) {
		return esCli, nil
	}
}

type mockedSubscriptionsClient struct {
	servicebus.SubscriptionsClient

	subs mockSubscriptions

	calledGet          bool
	calledCreateUpdate bool
	calledDelete       bool
}

// the fake client expects keys in the format <topic name>/<subscription name>
type mockSubscriptions map[string]azservicebus.SBSubscription

const testSubscriptionsClientDataKey = "subsClient"

func (c *mockedSubscriptionsClient) Get(ctx context.Context, resourceGroupName, namespaceName,
	topicName, subscriptionName string) (azservicebus.SBSubscription, error) {

	c.calledGet = true

	if len(c.subs) == 0 {
		return azservicebus.SBSubscription{}, notFoundAzureErr()
	}

	sub, ok := c.subs[topicName+"/"+subscriptionName]
	if !ok {
		return azservicebus.SBSubscription{}, notFoundAzureErr()
	}

	return sub, nil
}

func (c *mockedSubscriptionsClient) CreateOrUpdate(ctx context.Context, resourceGroupName, namespaceName,
	topicName, subscriptionName string, parameters azservicebus.SBSubscription) (azservicebus.SBSubscription, error) {

	c.calledCreateUpdate = true

	return azservicebus.SBSubscription{
		ID: to.Ptr(tSubscriptionID.String()),
	}, nil
}

func (c *mockedSubscriptionsClient) Delete(ctx context.Context, resourceGroupName, namespaceName,
	topicName, subscriptionName string) (autorest.Response, error) {

	c.calledDelete = true

	if len(c.subs) == 0 {
		return autorest.Response{}, notFoundAzureErr()
	}

	var err error
	if _, ok := c.subs[topicName+"/"+subscriptionName]; !ok {
		err = notFoundAzureErr()
	}

	return autorest.Response{}, err
}

const mockSubscriptionsDataKey = "subs"

// makeMockSubscriptions returns a mocked list of Subscriptions to be used as
// TableRow data.
func makeMockSubscriptions() map[string]interface{} {
	sub := azservicebus.SBSubscription{
		ID: to.Ptr(tSubscriptionID.String()),
	}

	// key format expected by mocked client impl
	subKey := tTopicID.ResourceName + "/" + tSubscriptionID.SubResourceName

	return map[string]interface{}{
		mockSubscriptionsDataKey: mockSubscriptions{
			subKey: sub,
		},
	}
}

// getMockSubscriptions gets mocked Subscriptions from the TableRow's data.
func getMockSubscriptions(tr *rt.TableRow) mockSubscriptions {
	hubs, ok := tr.OtherTestData[mockSubscriptionsDataKey]
	if !ok {
		return nil
	}
	return hubs.(mockSubscriptions)
}

func calledGetSubscription(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testSubscriptionsClientDataKey].(*mockedSubscriptionsClient)

		if expectCall && !cli.calledGet {
			t.Error("Did not call Get() on Subscription")
		}
		if !expectCall && cli.calledGet {
			t.Error("Unexpected call to Get() on Subscription")
		}
	}
}
func calledCreateUpdateSubscription(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testSubscriptionsClientDataKey].(*mockedSubscriptionsClient)

		if expectCall && !cli.calledCreateUpdate {
			t.Error("Did not call CreateOrUpdate() on Subscription")
		}
		if !expectCall && cli.calledCreateUpdate {
			t.Error("Unexpected call to CreateOrUpdate() on Subscription")
		}
	}
}
func calledDeleteSubscription(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testSubscriptionsClientDataKey].(*mockedSubscriptionsClient)

		if expectCall && !cli.calledDelete {
			t.Error("Did not call Delete() on Subscription")
		}
		if !expectCall && cli.calledDelete {
			t.Error("Unexpected call to Delete() on Subscription")
		}
	}
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

func createdSubsEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonSubscribed,
		"Created Subscription %q for Topic %q", tSubscriptionID.SubResourceName, &tTopicID)
}
func deletedSubsEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonUnsubscribed,
		"Deleted Subscription %q for Topic %q", tSubscriptionID.SubResourceName, &tTopicID)
}
func skippedDeleteSubsEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, ReasonUnsubscribed,
		"Subscription not found, skipping deletion")
}
func finalizedEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, "FinalizerUpdate", "Updated %q finalizers", tName)
}
