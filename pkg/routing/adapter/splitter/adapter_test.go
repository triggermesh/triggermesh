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

package splitter

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
	fakeinformer "github.com/triggermesh/triggermesh/pkg/client/generated/injection/informers/routing/v1alpha1/splitter/fake"
)

const (
	tCloudEventID     = "ce-abcd-0123"
	tCloudEventType   = "ce.test.type"
	tCloudEventSource = "ce.test.source"

	tNS = "test-namespace"
)

var tSplitter = struct {
	key  string
	path string
}{
	key:  tNS + "/splitter",
	path: "items",
}

var tCases = map[string]struct {
	input         string
	numberOfParts int
}{
	"event0": {
		input: `{
			"items":[
				{
					"id":5,
					"name":"foo"
				}
			  ]
			}`,
		numberOfParts: 1,
	},
	"event2": {
		input: `{
			"items":[
				{
					"id":5,
					"name":"foo"
				},{
					"id":10,
					"name":"bar"
				}
			  ]
			}`,
		numberOfParts: 2,
	},
	"event3": {
		input: `{
			"items":[
				{
					"id":5,
					"name":"foo"
				},{
					"id":10,
					"name":"bar"
				},{
					"id":15,
					"name":"baz"
				}
			  ]
			}`,
		numberOfParts: 3,
	},
	"event4": {
		input: `{
			"items":"not-an-array"
			}`,
		numberOfParts: 1,
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
		t.Run(name, func(t *testing.T) {
			splitterAddr := fmt.Sprintf("http://%s:%d/%s", ip, port, tSplitter.key)

			ce := newCloudEvent(t, tc.input)
			splitterResponse := sendCE(t, &ce, splitterAddr)
			assert.Equal(t, false, cloudevents.IsUndelivered(splitterResponse), "Could not deliver event to Splitter")

			// Simple check to verify if a number of returned new events
			// equals the number of items in the input event.
			counter := 0
			for range responses {
				counter++
				if counter == tc.numberOfParts {
					break
				}
			}
		})
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
	injection.Fake.RegisterClient(func(ctx context.Context, _ *rest.Config) context.Context {
		ctx, _ = fakeinjectionclient.With(ctx, newSplitter(t, tSplitter.key, tSplitter.path, sinkURI))
		return ctx
	})

	var infs []controller.Informer
	ctx, infs = injection.Fake.SetupInformers(ctx, &rest.Config{})
	err := controller.StartInformers(ctx.Done(), infs...)
	require.NoError(t, err)

	return ctx
}

func newSplitter(t *testing.T, key, path, addr string) *v1alpha1.Splitter {
	sinkURI, err := apis.ParseURL("http://" + addr)
	assert.NoError(t, err)

	return &v1alpha1.Splitter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  strings.Split(key, "/")[0],
			Name:       strings.Split(key, "/")[1],
			UID:        uuid.NewUUID(),
			Generation: 1,
		},
		Spec: v1alpha1.SplitterSpec{
			Path: path,
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
		receiver:       kncloudevents.NewHTTPMessageReceiver(port),
		sender:         sender,
		splitterLister: fakeinformer.Get(ctx).Lister().Splitters(tNS),
		logger:         logtesting.TestLogger(t),
	}
}

func setupSink(t *testing.T, ctx context.Context) (chan string, net.Addr) {
	c := make(chan string, 4)

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
