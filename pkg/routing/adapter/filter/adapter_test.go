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

package filter

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol"

	"knative.dev/eventing/pkg/kncloudevents"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	logtesting "knative.dev/pkg/logging/testing"

	common "github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/routing/v1alpha1"
	fakeinjectionclient "github.com/triggermesh/triggermesh/pkg/client/generated/injection/client/fake"
	fakeinformer "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/routing/v1alpha1/filter/fake"
)

const (
	tCloudEventID     = "ce-abcd-0123"
	tCloudEventType   = "ce.test.type"
	tCloudEventSource = "ce.test.source"

	tNS = "test-namespace"
)

type outputs struct {
	pass      bool // true - event must pass the filter, false - not
	wantError bool // filter response
}

var tFilters = []struct {
	key        string
	expression string
}{
	{
		key:        tNS + "/filter-1",
		expression: `$firstname.(string) == "bob"`,
	}, {
		key:        tNS + "/filter-2",
		expression: `$0.name.last.(string) == "smith" && $0.age.(int64) < 30`,
	}, {
		key:        tNS + "/malformed",
		expression: `hi!`,
	},
}

var tCases = map[string]struct {
	payload string
	filters map[string]outputs
}{
	"event0": {
		payload: `{"firstname":"alice","lastname":"smith"}`,
		filters: map[string]outputs{
			tNS + "/filter-1": {
				wantError: false,
				pass:      false,
			},
			tNS + "/filter-2": {
				wantError: false,
				pass:      false,
			},
		},
	},
	"event1": {
		payload: `{"firstname":"bob","lastname":"smith"}`,
		filters: map[string]outputs{
			tNS + "/filter-1": {
				wantError: false,
				pass:      true,
			},
			tNS + "/filter-2": {
				wantError: false,
				pass:      false,
			},
			tNS + "/malformed": {
				wantError: true,
				pass:      false,
			},
		},
	},
	"event2": {
		payload: `[{"name":{"first":"alice","last":"smith"},"age": 25}]`,
		filters: map[string]outputs{
			tNS + "/filter-1": {
				wantError: false,
				pass:      false,
			},
			tNS + "/filter-2": {
				wantError: false,
				pass:      true,
			},
		},
	},
	"event3": {
		payload: `[{"name":{"first":"bob","last":"smith"},"age": 42}]`,
		filters: map[string]outputs{
			tNS + "/filter-1": {
				wantError: false,
				pass:      false,
			},
			tNS + "/filter-2": {
				wantError: false,
				pass:      false,
			},
		},
	},
	"failingEvent": {
		payload: `{}`,
		filters: map[string]outputs{
			tNS + "/malformed": {
				wantError: true,
				pass:      false,
			},
			"wrongns/filter": {
				wantError: true,
				pass:      false,
			},
			"/filter": {
				wantError: true,
				pass:      false,
			},
			"bad/request/filter": {
				wantError: true,
				pass:      false,
			},
			tNS + "/missingfilter": {
				wantError: true,
				pass:      false,
			},
		},
	},
}

func TestAdapter(t *testing.T) {
	ip, port := testServerSocket(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	responses, sinkAddr := setupSink(t, ctx)
	ctx = setupFakeInformers(t, ctx, sinkAddr.String())

	h := newHandler(t, ctx, port)
	go func() {
		if err := h.Start(ctx); err != nil {
			assert.FailNow(t, "could not start test adapter", err)
		}
	}()
	<-h.receiver.Ready

	for name, tc := range tCases {
		for filter, outputs := range tc.filters {
			t.Run(fmt.Sprintf("%s(%s)", filter, name), func(t *testing.T) {
				filterAddr := fmt.Sprintf("http://%s:%d/%s", ip, port, filter)

				ce := newCloudEvent(t, tc.payload)
				filterResponse := sendCE(t, &ce, filterAddr)
				assert.Equal(t, outputs.wantError, cloudevents.IsNACK(filterResponse), "Filter didn't report bad request")

				select {
				case sinkResponse := <-responses:
					if outputs.pass {
						assert.Equal(t, tc.payload, sinkResponse)
					} else {
						t.Error("Unexpected event passed the filter")
					}
				default:
					if outputs.pass {
						t.Errorf("Expected event didn't pass the filter")
					}
				}
			})
		}
	}
}

func newCloudEvent(t *testing.T, data string) cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetID(tCloudEventID)
	event.SetType(tCloudEventType)
	event.SetSource(tCloudEventSource)
	err := event.SetData(cloudevents.ApplicationJSON, []byte(data))
	require.NoError(t, err)
	return event
}

func sendCE(t *testing.T, event *cloudevents.Event, sink string) protocol.Result {
	ctx := cloudevents.ContextWithTarget(context.Background(), sink)
	c, err := cloudevents.NewClientHTTP()
	require.NoError(t, err)

	return c.Send(ctx, *event)
}

func setupFakeInformers(t *testing.T, ctx context.Context, sinkURI string) context.Context {
	var filters []runtime.Object
	for _, f := range tFilters {
		filters = append(filters, newFilter(t, f.key, f.expression, sinkURI))
	}

	injection.Fake.RegisterClient(func(ctx context.Context, _ *rest.Config) context.Context {
		ctx, _ = fakeinjectionclient.With(ctx, filters...)
		return ctx
	})

	var infs []controller.Informer
	ctx, infs = injection.Fake.SetupInformers(ctx, &rest.Config{})
	err := controller.StartInformers(ctx.Done(), infs...)
	require.NoError(t, err)

	return ctx
}

func newFilter(t *testing.T, key, expression, addr string) *v1alpha1.Filter {
	sinkURI, err := apis.ParseURL("http://" + addr)
	assert.NoError(t, err)

	return &v1alpha1.Filter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  strings.Split(key, "/")[0],
			Name:       strings.Split(key, "/")[1],
			UID:        uuid.NewUUID(),
			Generation: 1,
		},
		Spec: v1alpha1.FilterSpec{
			Expression: expression,
		},
		Status: common.Status{
			SourceStatus: duckv1.SourceStatus{
				SinkURI: sinkURI,
			},
		},
	}
}

func newHandler(t *testing.T, ctx context.Context, port int) *Handler {
	sender, err := kncloudevents.NewHTTPMessageSenderWithTarget("")
	assert.NoError(t, err)

	return &Handler{
		receiver:     kncloudevents.NewHTTPMessageReceiver(port),
		sender:       sender,
		filterLister: fakeinformer.Get(ctx).Lister().Filters(tNS),
		logger:       logtesting.TestLogger(t),
		expressions:  newExpressionStorage(),
	}
}

func setupSink(t *testing.T, ctx context.Context) (chan string, net.Addr) {
	c := make(chan string, 1)

	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		c <- string(body)
		w.WriteHeader(http.StatusOK)
	}))

	go func() {
		<-ctx.Done()
		sink.Close()
		close(c)
	}()

	return c, sink.Listener.Addr()
}

// testServerSocket returns a local IP and port where HTTP server can safely listen on.
func testServerSocket(t *testing.T) (net.IP, int) {
	t.Helper()

	// hack: Create a net.Listener on a random port, close it, and return its address.
	// This guarantees that the address:port combination is free on the local host.
	// See http/httptest.newLocalListener()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			t.Fatalf("failed to listen on a port: %v", err)
		}
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).IP, l.Addr().(*net.TCPAddr).Port
}
