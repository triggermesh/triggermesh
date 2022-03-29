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

package awssnssource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgotesting "k8s.io/client-go/testing"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/reconciler"
	rt "knative.dev/pkg/reconciler/testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/awssnssource"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/awssnssource/status"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/router"
	adaptesting "github.com/triggermesh/triggermesh/pkg/sources/adapter/testing"
	snsclient "github.com/triggermesh/triggermesh/pkg/sources/client/sns"
	eventtesting "github.com/triggermesh/triggermesh/pkg/sources/testing/event"
)

func TestReconcile(t *testing.T) {
	testCases := rt.TableTest{
		// Creation/Deletion

		{
			Name: "Initial handler registration",
			Key:  tKey,
			Ctx:  statusMockClockContext(),
			Objects: []runtime.Object{
				newEventSource(),
			},
			WantPatches: []clientgotesting.PatchActionImpl{
				handlerRegisteredPatch(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				isRegistered,
			},
		},
		{
			Name: "Source deleted",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventSource(deleted),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				isDeregistered,
			},
		},

		// Lifecycle

		{
			Name: "Handler previously registered",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventSource(registered),
			},
			WantPatches: []clientgotesting.PatchActionImpl{
				// no patch
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				isRegistered,
			},
		},

		// Errors

		{
			Name: "Sink not ready",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventSource(noSink),
			},
			WantEvents: []string{
				sinkMissingEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				isDeregistered,
			},
			WantErr: true,
		},
		{
			Name: "Error fetching credentials",
			Key:  tKey,
			Ctx:  failingSNSClientGetterContext(),
			Objects: []runtime.Object{
				newEventSource(),
			},
			WantEvents: []string{
				failGetSNSClientEvent(),
			},
			PostConditions: []func(*testing.T, *rt.TableRow){
				isDeregistered,
			},
			WantErr: true,
		},

		// Edge cases

		{
			Name:    "Reconcile a non-existing object",
			Key:     tKey,
			Objects: nil,
			WantErr: false,
		},
	}

	ctor := reconcilerCtor()

	testCases.Test(t, adaptesting.MakeFactory(ctor))
}

// reconcilerCtor returns a Ctor for a AWSSNSSource Reconciler.
func reconcilerCtor() adaptesting.Ctor {
	return func(t *testing.T, ctx context.Context, tr *rt.TableRow, ls *adaptesting.Listers) controller.Reconciler {

		srcClienset := fakeinjectionclient.Get(ctx)

		a := &adapter{
			logger:   logtesting.TestLogger(t),
			ceClient: adaptertest.NewTestClient(),
			snsCg:    snsClientGetterFromContext(ctx),
			router:   &router.Router{},
			statusPatcher: status.NewPatcher(tComponent,
				srcClienset.SourcesV1alpha1().AWSSNSSources(tNs),
			),
		}

		// inject adapter into test data so that table tests can perform
		// assertions on it
		if tr.OtherTestData == nil {
			tr.OtherTestData = make(map[string]interface{}, 1)
		}
		tr.OtherTestData[testAdapterDataKey] = a

		r := &Reconciler{
			adapter: a,
		}

		return reconcilerv1alpha1.NewReconciler(ctx, logging.FromContext(ctx),
			srcClienset, ls.GetAWSSNSSourceLister(),
			controller.GetEventRecorder(ctx), r)
	}
}

const (
	tNs      = "testns"
	tName    = "test"
	tKey     = tNs + "/" + tName
	tURLPath = "/" + tKey

	tComponent = "test-component"

	tTopicARN = "arn:aws:sns:us-fake-0:123456789012:MyTopic"
)

var tSinkURI = &apis.URL{
	Scheme: "http",
	Host:   "default.default.svc.example.com",
	Path:   "/",
}

/* Event sources */

// sourceOption is a functional option for an event source.
type sourceOption func(*v1alpha1.AWSSNSSource)

// newEventSource returns a test source object with pre-filled attributes.
func newEventSource(opts ...sourceOption) *v1alpha1.AWSSNSSource {
	src := &v1alpha1.AWSSNSSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
		},
		Status: v1alpha1.AWSSNSSourceStatus{
			EventSourceStatus: v1alpha1.EventSourceStatus{
				SourceStatus: duckv1.SourceStatus{
					SinkURI: tSinkURI,
				},
			},
		},
	}

	// *reconcilerImpl.Reconcile calls this method before any reconciliation loop. Calling it here ensures that the
	// object is initialized in the same manner, and prevents tests from wrongly reporting unexpected status updates.
	reconciler.PreProcessReconcile(context.Background(), src)

	for _, opt := range opts {
		opt(src)
	}

	return src
}

// registered sets the HandlerRegistered status condition.
func registered(src *v1alpha1.AWSSNSSource) {
	src.Status.Conditions = append(src.Status.Conditions, apis.Condition{
		Type:     v1alpha1.AWSSNSConditionHandlerRegistered,
		Status:   corev1.ConditionTrue,
		Severity: apis.ConditionSeverityInfo,
		// LastTransitionTime can be omitted, it is excluded from the
		// comparison if the above fields already match
	})
}

// noSink ensures the sink URI is absent from the source's status.
func noSink(src *v1alpha1.AWSSNSSource) {
	src.Status.SinkURI = nil
}

// deleted marks the source as deleted.
func deleted(src *v1alpha1.AWSSNSSource) {
	t := metav1.Unix(0, 0)
	src.SetDeletionTimestamp(&t)
}

/* Events */

func sinkMissingEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, ReasonSourceNotReady,
		"Event sink URL wasn't resolved yet. Skipping adapter configuration")
}
func failGetSNSClientEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, "InternalError", "registering HTTP handler: "+
		"obtaining SNS client: assert.AnError general error for testing")
}

/* Patches */

func handlerRegisteredPatch() clientgotesting.PatchActionImpl {
	return clientgotesting.PatchActionImpl{
		Name:      tName,
		PatchType: types.JSONPatchType,
		Patch: []byte(`[{` +
			`"op":"add",` +
			`"path":"/status/conditions/1",` +
			`"value":{` +
			`"lastTransitionTime":"1970-01-01T00:00:00Z",` +
			`"severity":"Info",` +
			`"status":"True",` +
			`"type":"` + v1alpha1.AWSSNSConditionHandlerRegistered + `"` +
			`}` +
			`}]`,
		),
	}
}

/* Test contexts */

// fakeClock returns a time that is always the 0 epoch.
type fakeClock struct{}

// Now implements status.Clock.
func (*fakeClock) Now() apis.VolatileTime {
	return apis.VolatileTime{
		Inner: metav1.Unix(0, 0),
	}
}

func statusMockClockContext() context.Context {
	return status.WithClock(context.Background(), &fakeClock{})
}

var snscgKey struct{}

var defaultSNSClientGetter = staticClientGetter(&mockedSNSClient{})

// failingClientGetter is a sns.ClientGetter that always returns an error.
type failingClientGetter struct{}

// Get implements sns.ClientGetter.
func (*failingClientGetter) Get(*v1alpha1.AWSSNSSource) (snsclient.Client, error) {
	return nil, assert.AnError
}

var _ snsclient.ClientGetter = (*failingClientGetter)(nil)

// failingSNSClientGetterContext returns a context with a failingClientGetter
// attached.
func failingSNSClientGetterContext() context.Context {
	return context.WithValue(context.Background(), snscgKey, &failingClientGetter{})
}

// snsClientGetterFromContext returns the sns.ClientGetter associated with the
// context, or a default mocked client getter as a fall back.
func snsClientGetterFromContext(ctx context.Context) snsclient.ClientGetter {
	if cg, ok := ctx.Value(snscgKey).(snsclient.ClientGetter); ok {
		return cg
	}
	return defaultSNSClientGetter
}

/* Adapter */

const testAdapterDataKey = "adapter"

// staticClientGetter transforms the given client interface into a
// ClientGetter.
func staticClientGetter(cli snsclient.Client) snsclient.ClientGetterFunc {
	return func(*v1alpha1.AWSSNSSource) (snsclient.Client, error) {
		return cli, nil
	}
}

// mockedSNSClient always succeeds subscription confirmations.
type mockedSNSClient struct {
	snsclient.Client
}

func (c *mockedSNSClient) ConfirmSubscription(*sns.ConfirmSubscriptionInput) (*sns.ConfirmSubscriptionOutput, error) {
	return &sns.ConfirmSubscriptionOutput{
		SubscriptionArn: aws.String(tTopicARN + "/0123456789"),
	}, nil
}

// isRegistered verifies that the test endpoint responds with a status code
// different from NotFound.
func isRegistered(t *testing.T, tr *rt.TableRow) {
	a := tr.OtherTestData[testAdapterDataKey].(*adapter)

	resp := probeHandler(t, a, tURLPath)
	assert.NotEqual(t, http.StatusNotFound, resp.Code, "Expected handler hit")
}

// isDeregistered verifies that the test endpoint responds with a NotFound
// status code.
func isDeregistered(t *testing.T, tr *rt.TableRow) {
	a := tr.OtherTestData[testAdapterDataKey].(*adapter)

	resp := probeHandler(t, a, tURLPath)
	assert.Equal(t, http.StatusNotFound, resp.Code, "Expected no handler")
}

// probeHandler probes the given HTTP handler at the selected URL path and
// returns the recorded response.
func probeHandler(t *testing.T, h http.Handler, urlPath string) *httptest.ResponseRecorder {
	t.Helper()

	req, err := http.NewRequest(http.MethodHead, urlPath, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	return rr
}
