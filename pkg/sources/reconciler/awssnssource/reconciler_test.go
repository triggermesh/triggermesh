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

package awssnssource

import (
	"context"
	"testing"

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
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sns"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/awssnssource"
	snsclient "github.com/triggermesh/triggermesh/pkg/sources/client/sns"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/common"
	. "github.com/triggermesh/triggermesh/pkg/sources/reconciler/testing"
	"github.com/triggermesh/triggermesh/pkg/sources/routing"
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

// reconcilerCtor returns a Ctor for a AWSSNSSource Reconciler.
func reconcilerCtor(cfg *adapterConfig) Ctor {
	return func(t *testing.T, ctx context.Context, tr *rt.TableRow, ls *Listers) controller.Reconciler {
		snsCli := &mockedSNSClient{
			subscriptions: getMockSubscriptionsPages(tr),
		}

		// inject client into test data so that table tests can perform
		// assertions on it
		if tr.OtherTestData == nil {
			tr.OtherTestData = make(map[string]interface{}, 1)
		}
		tr.OtherTestData[testClientDataKey] = snsCli

		r := &Reconciler{
			base:       NewTestServiceReconciler(ctx, ls),
			adapterCfg: cfg,
			srcLister:  ls.GetAWSSNSSourceLister().AWSSNSSources,
			snsCg:      staticClientGetter(snsCli),
		}

		return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
			fakeinjectionclient.Get(ctx), ls.GetAWSSNSSourceLister(),
			controller.GetEventRecorder(ctx), r)
	}
}

// newEventSource returns a test source object with a minimal set of pre-filled attributes.
func newEventSource() *v1alpha1.AWSSNSSource {
	src := &v1alpha1.AWSSNSSource{
		Spec: v1alpha1.AWSSNSSourceSpec{
			ARN: NewARN(sns.ServiceName, "triggermeshtest"),
			SubscriptionAttributes: map[string]*string{
				"DeliveryPolicy": aws.String(`{"healthyRetryPolicy":{"numRetries":5}}`),
			},
			Credentials: v1alpha1.AWSSecurityCredentials{
				AccessKeyID: v1alpha1.ValueFromField{
					ValueFromSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "keyId",
					},
				},
				SecretAccessKey: v1alpha1.ValueFromField{
					ValueFromSecret: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "secret",
					},
				},
			},
		},
	}

	// assume finalizer is already set to prevent the generated reconciler
	// from generating an extra Patch action
	src.Finalizers = []string{sources.AWSSNSSourceResource.String()}

	Populate(src)

	return src
}

// adapterBuilder returns a slim Reconciler containing only the fields accessed
// by r.BuildAdapter().
func adapterBuilder(cfg *adapterConfig) common.AdapterServiceBuilder {
	return &Reconciler{
		adapterCfg: cfg,
	}
}

// TestReconcileSubscription contains tests specific to the SNS source.
func TestReconcileSubscription(t *testing.T) {
	testCases := rt.TableTest{
		// Regular lifecycle

		{
			Name:          "Not yet subscribed",
			Key:           tKey,
			OtherTestData: makeMockSubscriptionsPages(false),
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
				subscribedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledSubscribe(true),
			},
		},
		{
			Name:          "Already subscribed",
			Key:           tKey,
			OtherTestData: makeMockSubscriptionsPages(true),
			Objects: []runtime.Object{
				newReconciledSource(subscribed),
				newReconciledServiceAccount(),
				newReconciledRoleBinding(),
				newReconciledAdapter(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledSubscribe(false),
			},
		},

		// Finalization

		{
			Name:          "Deletion while subscribed",
			Key:           tKey,
			OtherTestData: makeMockSubscriptionsPages(true),
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
				finalizedEvent(),
				unsubscribedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledUnsubscribe(true),
			},
		},
		{
			Name:          "Deletion while not subscribed",
			Key:           tKey,
			OtherTestData: makeMockSubscriptionsPages(false),
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
				notEnoughInformationUnsubscribeEvent(),
				finalizedEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				calledUnsubscribe(false),
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

	tAdapterURI = &apis.URL{
		Scheme: "http",
		Host:   "public.example.com",
		Path:   "/",
	}
)

/* Source and receive adapter */

// sourceOption is a functional option for an event source.
type sourceOption func(*v1alpha1.AWSSNSSource)

// newReconciledSource returns a test event source object that is identical to
// what ReconcileKind generates.
func newReconciledSource(opts ...sourceOption) *v1alpha1.AWSSNSSource {
	src := newEventSource()

	// assume the sink URI is resolved
	src.Spec.Sink.Ref = nil
	src.Spec.Sink.URI = tSinkURI

	// assume status conditions are already set to True to ensure
	// ReconcileKind is a no-op
	status := src.GetStatusManager()
	status.MarkSink(tSinkURI)
	status.PropagateServiceAvailability(newReconciledAdapter())
	status.SetRoute(routing.URLPath(src))

	for _, opt := range opts {
		opt(src)
	}

	return src
}

var (
	tTopicARN = NewARN(sns.ServiceName, "triggermeshtest")
	tSubARN   = NewARN(sns.ServiceName, "triggermeshtest/0123456789")
)

// subscribed sets the Subscribed status condition to True and reports the ARN
// of the SNS subscription in the source's status.
func subscribed(src *v1alpha1.AWSSNSSource) {
	src.Status.MarkSubscribed(tSubARN.String())
}

// deleted marks the source as deleted.
func deleted(src *v1alpha1.AWSSNSSource) {
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
func newReconciledAdapter() *servingv1.Service {
	adapter := adapterBuilder(adapterCfg).BuildAdapter(newEventSource(), tSinkURI)

	common.OwnByServiceAccount(adapter, NewServiceAccount(newEventSource())())

	adapter.Status.SetConditions(apis.Conditions{{
		Type:   v1alpha1.ConditionReady,
		Status: corev1.ConditionTrue,
	}})
	adapter.Status.URL = tAdapterURI

	return adapter
}

/* SNS client */

// staticClientGetter transforms the given client interface into a
// ClientGetter.
func staticClientGetter(cli snsclient.Client) snsclient.ClientGetterFunc {
	return func(*v1alpha1.AWSSNSSource) (snsclient.Client, error) {
		return cli, nil
	}
}

const testClientDataKey = "client"

type mockSubscriptionsPages map[ /*token*/ *string][]*sns.Subscription

type mockedSNSClient struct {
	snsclient.Client

	subscriptions mockSubscriptionsPages

	calledSubscribe   bool
	calledUnsubscribe bool
}

func (c *mockedSNSClient) SubscribeWithContext(aws.Context, *sns.SubscribeInput,
	...request.Option) (*sns.SubscribeOutput, error) {

	c.calledSubscribe = true

	return &sns.SubscribeOutput{
		SubscriptionArn: aws.String(tSubARN.String()),
	}, nil
}

func (c *mockedSNSClient) UnsubscribeWithContext(aws.Context, *sns.UnsubscribeInput,
	...request.Option) (*sns.UnsubscribeOutput, error) {

	c.calledUnsubscribe = true

	return &sns.UnsubscribeOutput{}, nil
}

var page2Token = aws.String("page2token")

func (c *mockedSNSClient) ListSubscriptionsByTopicWithContext(_ aws.Context, in *sns.ListSubscriptionsByTopicInput,
	_ ...request.Option) (*sns.ListSubscriptionsByTopicOutput, error) {

	if len(c.subscriptions) == 0 {
		return &sns.ListSubscriptionsByTopicOutput{}, nil
	}

	var nextToken *string
	if in.NextToken == nil {
		nextToken = page2Token
	}

	return &sns.ListSubscriptionsByTopicOutput{
		Subscriptions: c.subscriptions[in.NextToken],
		NextToken:     nextToken,
	}, nil
}

const mockSubscriptionsPagesDataKey = "subpages"

// makeMockSubscriptionsPages returns mocked pages of SNS Subscriptions to be
// used as TableRow data.
func makeMockSubscriptionsPages(subExists bool) map[string]interface{} {
	pages := make(mockSubscriptionsPages, 2)

	var wrongSubURL = aws.String("http://not-my-sub.example.com")
	var wrongSubARN = aws.String("aws:sns:not:my:sub")

	var okSubURL = aws.String(tAdapterURI.String() + tNs + "/" + tName)
	var okSubARN = aws.String(tSubARN.String())

	// first page, retrieved without NextToken
	pages[new(string)] = []*sns.Subscription{
		{Endpoint: wrongSubURL, SubscriptionArn: wrongSubARN},
		{Endpoint: wrongSubURL, SubscriptionArn: wrongSubARN},
		{Endpoint: wrongSubURL, SubscriptionArn: wrongSubARN},
	}

	// second page, retrieved with NextToken
	pages[page2Token] = []*sns.Subscription{
		{Endpoint: wrongSubURL, SubscriptionArn: wrongSubARN},
		{Endpoint: wrongSubURL, SubscriptionArn: wrongSubARN},
		{Endpoint: wrongSubURL, SubscriptionArn: wrongSubARN},
	}

	// inject the expected Subscription in the second page if requested, at
	// a non-zero index to ensure pagination is handled correctly
	if subExists {
		pages[page2Token][1].Endpoint = okSubURL
		pages[page2Token][1].SubscriptionArn = okSubARN
	}

	return map[string]interface{}{
		mockSubscriptionsPagesDataKey: pages,
	}
}

// getMockSubscriptionsPages gets mocked pages of SNS Subscriptions from the
// TableRow's data.
func getMockSubscriptionsPages(tr *rt.TableRow) mockSubscriptionsPages {
	pages, ok := tr.OtherTestData[mockSubscriptionsPagesDataKey]
	if !ok {
		return nil
	}
	return pages.(mockSubscriptionsPages)
}

func calledSubscribe(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testClientDataKey].(*mockedSNSClient)

		if expectCall && !cli.calledSubscribe {
			t.Error("Did not call Subscribe()")
		}
		if !expectCall && cli.calledSubscribe {
			t.Error("Unexpected call to Subscribe()")
		}
	}
}

func calledUnsubscribe(expectCall bool) func(*testing.T, *rt.TableRow) {
	return func(t *testing.T, tr *rt.TableRow) {
		cli := tr.OtherTestData[testClientDataKey].(*mockedSNSClient)

		if expectCall && !cli.calledUnsubscribe {
			t.Error("Did not call Unsubscribe()")
		}
		if !expectCall && cli.calledUnsubscribe {
			t.Error("Unexpected call to Unsubscribe()")
		}
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

func subscribedEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonSubscribed, "Subscribed to SNS topic %q", tTopicARN)
}
func unsubscribedEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, ReasonUnsubscribed, "Unsubscribed from SNS topic %q", tTopicARN)
}

func notEnoughInformationUnsubscribeEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, ReasonFailedUnsubscribe, "SNS status information incomplete, skipping finalization")
}

func finalizedEvent() string {
	return eventtesting.Eventf(corev1.EventTypeNormal, "FinalizerUpdate", "Updated %q finalizers", tName)
}
