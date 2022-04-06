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

package zendesksource

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

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/reconciler"
	rt "knative.dev/pkg/reconciler/testing"

	commonv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	reconcilerv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/injection/reconciler/sources/v1alpha1/zendesksource"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/router"
	adaptesting "github.com/triggermesh/triggermesh/pkg/sources/adapter/testing"
	"github.com/triggermesh/triggermesh/pkg/sources/secret"
	eventtesting "github.com/triggermesh/triggermesh/pkg/testing/event"
)

func TestReconcile(t *testing.T) {
	testCases := rt.TableTest{
		// Creation/Deletion

		{
			Name: "Handler registration",
			Key:  tKey,
			Objects: []runtime.Object{
				newEventSource(),
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
			Ctx:  failingSecretGetterContext(),
			Objects: []runtime.Object{
				newEventSource(),
			},
			WantEvents: []string{
				failWebhookCredentialsEvent(),
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

// reconcilerCtor returns a Ctor for a ZendeskSource Reconciler.
func reconcilerCtor() adaptesting.Ctor {
	return func(t *testing.T, ctx context.Context, tr *rt.TableRow, ls *adaptesting.Listers) controller.Reconciler {

		a := &adapter{
			logger:     logtesting.TestLogger(t),
			ceClient:   adaptertest.NewTestClient(),
			secrGetter: secretGetterFromContext(ctx),
			router:     &router.Router{},
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
			fakeinjectionclient.Get(ctx), ls.GetZendeskSourceLister(),
			controller.GetEventRecorder(ctx), r)
	}
}

const (
	tNs      = "testns"
	tName    = "test"
	tKey     = tNs + "/" + tName
	tURLPath = "/" + tKey
)

var tSinkURI = &apis.URL{
	Scheme: "http",
	Host:   "default.default.svc.example.com",
	Path:   "/",
}

/* Event sources */

// sourceOption is a functional option for an event source.
type sourceOption func(*v1alpha1.ZendeskSource)

// newEventSource returns a test source object with pre-filled attributes.
func newEventSource(opts ...sourceOption) *v1alpha1.ZendeskSource {
	src := &v1alpha1.ZendeskSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tNs,
			Name:      tName,
		},
		Status: v1alpha1.ZendeskSourceStatus{
			Status: commonv1alpha1.Status{
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

// noSink ensures the sink URI is absent from the source's status.
func noSink(src *v1alpha1.ZendeskSource) {
	src.Status.SinkURI = nil
}

// deleted marks the source as deleted.
func deleted(src *v1alpha1.ZendeskSource) {
	t := metav1.Unix(0, 0)
	src.SetDeletionTimestamp(&t)
}

/* Events */

func sinkMissingEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, ReasonSourceNotReady,
		"Event sink URL wasn't resolved yet. Skipping adapter configuration")
}
func failWebhookCredentialsEvent() string {
	return eventtesting.Eventf(corev1.EventTypeWarning, "InternalError", "registering HTTP handler: "+
		"obtaining webhook secret: assert.AnError general error for testing")
}

/* Test contexts */

var secrGetterKey struct{}

type mockedSecretGetter struct {
	fail bool
}

var _ secret.Getter = (*mockedSecretGetter)(nil)

// Get implements secret.Getter.
func (sg *mockedSecretGetter) Get(refs ...commonv1alpha1.ValueFromField) (secret.Secrets, error) {
	if sg.fail {
		return nil, assert.AnError
	}

	const fakeVal = "fake"

	secrets := make(secret.Secrets, len(refs))

	for i := range refs {
		secrets[i] = fakeVal
	}

	return secrets, nil
}

// failingSecretGetterContext returns a context with a mocked secret.Getter
// that always fails.
func failingSecretGetterContext() context.Context {
	return context.WithValue(context.Background(), secrGetterKey,
		&mockedSecretGetter{fail: true},
	)
}

// secretGetterFromContext returns the secret.Getter associated with the
// context, or a default mocked Getter as a fall back.
func secretGetterFromContext(ctx context.Context) secret.Getter {
	if sg, ok := ctx.Value(secrGetterKey).(secret.Getter); ok {
		return sg
	}
	return &mockedSecretGetter{}
}

/* Adapter */

const testAdapterDataKey = "adapter"

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
