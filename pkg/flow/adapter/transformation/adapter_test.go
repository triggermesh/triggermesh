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

package transformation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"

	logtesting "knative.dev/pkg/logging/testing"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/storage"
)

var availableTransformations = []v1alpha1.Transform{
	{Operation: "add"},
	{Operation: "store"},
	{Operation: "shift"},
	{Operation: "delete"},
}

func TestStart(t *testing.T) {
	pipeline, err := newPipeline(availableTransformations, storage.New())
	assert.NoError(t, err)

	ceClient, err := cloudevents.NewClientHTTP()
	assert.NoError(t, err)

	a := &adapter{
		ContextPipeline: pipeline,
		DataPipeline:    pipeline,

		client: ceClient,
		logger: logtesting.TestLogger(t),
	}

	errChan := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer close(errChan)
		errChan <- a.Start(ctx)
	}()

	cancel()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer waitCancel()

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-waitCtx.Done():
		t.Error("Start() shutdown by context wait timeout")
	}
}

func setData(t *testing.T, event cloudevents.Event, data interface{}) cloudevents.Event {
	assert.NoError(t, event.SetData(cloudevents.ApplicationJSON, data))
	return event
}

func setExtensions(event cloudevents.Event, extensions map[string]interface{}) cloudevents.Event {
	for k, v := range extensions {
		event.SetExtension(k, v)
	}
	return event
}

func newEvent() cloudevents.Event {
	emptyV1Event := cloudevents.NewEvent(cloudevents.VersionV1)
	emptyV1Event.SetDataContentType(cloudevents.ApplicationJSON)
	emptyV1Event.SetID("123")
	emptyV1Event.SetSource("test")
	emptyV1Event.SetType("test")
	return emptyV1Event
}

func TestReceiveAndTransform(t *testing.T) {
	testCases := []struct {
		name                    string
		originalEvent           cloudevents.Event
		expectedEventData       string
		expectedEventExtensions string
		data                    []v1alpha1.Transform
		context                 []v1alpha1.Transform
	}{
		{
			name: "Conditional Add 1",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`{"name":"John"}`)),
			expectedEventData: `{"id":"John"}`,
			data: []v1alpha1.Transform{
				{
					Operation: "store",
					Paths: []v1alpha1.Path{
						{
							Key:   "$name",
							Value: "name",
						}, {
							Key:   "$surname",
							Value: "surname",
						},
					},
				}, {
					Operation: "delete",
					Paths: []v1alpha1.Path{
						{
							Key: "",
						},
					},
				}, {
					Operation: "add",
					Paths: []v1alpha1.Path{
						{
							Key:   "id",
							Value: `$name(.$surname)`,
						},
					},
				},
			},
		}, {
			name: "Conditional Add 2",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`{"day":"01","month":"01","year":"1970"}`)),
			expectedEventData: `{"date":"01/01/1970/1970"}`,
			data: []v1alpha1.Transform{
				{
					Operation: "store",
					Paths: []v1alpha1.Path{
						{
							Key:   "$day",
							Value: "day",
						}, {
							Key:   "$month",
							Value: "month",
						}, {
							Key:   "$year",
							Value: "year",
						},
					},
				}, {
					Operation: "delete",
					Paths: []v1alpha1.Path{
						{
							Key: "",
						},
					},
				}, {
					Operation: "add",
					Paths: []v1alpha1.Path{
						{
							Key:   "date",
							Value: `($day)(/$month)(/$year)/$year`,
						},
					},
				},
			},
		}, {
			name: "Custom separator",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`{"field.1":"value"}`)),
			expectedEventData: `{"nested":{"field.1":"value"},"not.nested":"foo"}`,
			data: []v1alpha1.Transform{
				{
					Operation: "shift",
					Paths: []v1alpha1.Path{
						{
							Key:       "field.1:nested/field.1",
							Separator: "/",
						},
					},
				}, {
					Operation: "add",
					Paths: []v1alpha1.Path{
						{
							Key:       "not.nested",
							Value:     "foo",
							Separator: ":",
						},
					},
				},
			},
		}, {
			name: "Add operation",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`{"foo":"bar","blah":[{"bleh":"huh?"}]}`)),
			expectedEventData: `{"blah":[{"bleh":"no"},null,{"foo":"42"}],"foo":"baz","message":"Hello World!","object":{"message":"hey","slice":[null,"sup"]}}`,
			data: []v1alpha1.Transform{
				{
					Operation: "add",
					Paths: []v1alpha1.Path{
						{
							// add key
							Key:   "message",
							Value: "Hello World!",
						}, {
							// add object
							Key:   "object.message",
							Value: "hey",
						}, {
							// append array value
							Key:   "object.slice[1]",
							Value: "sup",
						}, {
							// append object to an array
							Key:   "blah[2].foo",
							Value: "42",
						}, {
							// overwrite object in the array
							Key:   "blah[0].bleh",
							Value: "no",
						}, {
							// overwrite original key
							Key:   "foo",
							Value: "baz",
						},
					},
				},
			},
		}, {
			name: "Delete operation",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`{"key1":"value1","key2":"value2","key3":"value3","key4":"value4"}`)),
			expectedEventData: `{"key2":"value2"}`,
			data: []v1alpha1.Transform{
				{
					Operation: "delete",
					Paths: []v1alpha1.Path{
						{
							// just delete the key
							Key:   "key1",
							Value: "",
						}, {
							// actual value is not equal to the filter, ignore
							Key:   "key2",
							Value: "wrong filter",
						}, {
							// actual and expected values are equal, delete
							Key:   "key3",
							Value: "value3",
						}, {
							// delete all keys with this value
							Value: "value4",
						},
					},
				},
			},
		}, {
			name: "Delete operation, wipe payload",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`{"key1":"value1"}`)),
			expectedEventData: "null",
			data: []v1alpha1.Transform{
				{
					// wipe event payload
					Operation: "delete",
					Paths: []v1alpha1.Path{
						{
							Key: "",
						},
					},
				},
			},
		}, {
			name: "Shift operation",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`{"key1":"value1","object":{"key2":"value2"}}`)),
			expectedEventData: `{"key1":"value1","object":{},"okey":"value2"}`,
			data: []v1alpha1.Transform{
				{
					Operation: "shift",
					Paths: []v1alpha1.Path{
						{
							Key: "object.key2:okey",
						}, {
							Key:   "key1:key2",
							Value: "wrong filter",
						},
					},
				},
			},
		}, {
			name: "Shift full body",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`[{"key1":"value1"}]`)),
			expectedEventData: `{"body":[{"key1":"value1"}]}`,
			data: []v1alpha1.Transform{
				{
					Operation: "shift",
					Paths: []v1alpha1.Path{
						{
							Key: ".:body",
						},
					},
				},
			},
		}, {
			name: "Store operation",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`{"key1":"value1","object":{"foo":"bar"}}`)),
			expectedEventData: `{"key1":"bar","object":{"foo":"value1"}}`,
			data: []v1alpha1.Transform{
				{
					Operation: "store",
					Paths: []v1alpha1.Path{
						{
							Key:   "$var1",
							Value: "key1",
						}, {
							Key:   "$var2",
							Value: "object.foo",
						},
					},
				}, {
					Operation: "add",
					Paths: []v1alpha1.Path{
						{
							Key:   "key1",
							Value: "$var2",
						}, {
							Key:   "object.foo",
							Value: "$var1",
						},
					},
				},
			},
		}, {
			name: "Store full body",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`[{"key1":"value1"}]`)),
			expectedEventData: `{"body":[{"key1":"value1"}]}`,
			data: []v1alpha1.Transform{
				{
					Operation: "store",
					Paths: []v1alpha1.Path{
						{
							Key:   "$body",
							Value: ".",
						},
					},
				}, {
					Operation: "delete",
					Paths: []v1alpha1.Path{
						{
							Key: "",
						},
					},
				}, {
					Operation: "add",
					Paths: []v1alpha1.Path{
						{
							Key:   "body",
							Value: "$body",
						},
					},
				},
			},
		}, {
			name: "Parse operation",
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`{"key1":"value1","key2":[{"key3":"value3","strJSON":"{\"foo\":123,\"bar\":\"value2\",\"baz\":[\"one\",\"two\",\"three\"]}"}]}`)),
			expectedEventData: `{"key1":"value1","key2":[{"key3":"value3","strJSON":{"bar":"value2","baz":["one","two","three"],"foo":123}}]}`,
			data: []v1alpha1.Transform{
				{
					Operation: "parse",
					Paths: []v1alpha1.Path{
						{
							Key:   "key2[0].strJSON",
							Value: "json",
						},
					},
				},
			},
		}, {
			name: "Transform extensions",
			originalEvent: setExtensions(newEvent(),
				map[string]interface{}{"ext1": "value1", "ext2": 2}),
			expectedEventExtensions: `{"ext2":2,"ext3":"value3"}`,
			context: []v1alpha1.Transform{
				{
					Operation: "add",
					Paths: []v1alpha1.Path{
						{
							Key:   "Extensions.ext3",
							Value: "value3",
						},
					},
				},
				{
					Operation: "delete",
					Paths: []v1alpha1.Path{
						{
							Key: "Extensions.ext1",
						},
					},
				},
			},
		}, {
			name: "Ignore errors", //ensure that errored transformation won't affect the event
			originalEvent: setData(t, newEvent(),
				json.RawMessage(`{"key1":"value1","object":{"foo":"bar"}}`)),
			expectedEventData: `{"key1":"value1","object":{"foo":"bar"}}`,
			data: []v1alpha1.Transform{
				{
					Operation: "parse",
					Paths: []v1alpha1.Path{
						{
							Key:   "bad-value-type",
							Value: "jnos",
						}, {
							Key:   "Non-existing-key",
							Value: "json",
						},
					},
				}, {
					Operation: "shift",
					Paths: []v1alpha1.Path{
						{
							Key: "Non-existing-key:key1",
						}, {
							Key: "object.non-existing-key:key2",
						},
					},
				}, {
					Operation: "delete",
					Paths: []v1alpha1.Path{
						{
							Key: "Non-existing-key",
						},
					},
				}, {
					Operation: "store",
					Paths: []v1alpha1.Path{
						{
							Key:   "$var1",
							Value: "Non-existing-key",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			storage := storage.New()
			dataPipeline, err := newPipeline(tc.data, storage)
			assert.NoError(t, err)

			contextPipeline, err := newPipeline(tc.context, storage)
			assert.NoError(t, err)

			a := &adapter{
				DataPipeline:    dataPipeline,
				ContextPipeline: contextPipeline,
				logger:          logtesting.TestLogger(t),
			}

			transformedEvent, err := a.applyTransformations(tc.originalEvent)
			assert.NoError(t, err)

			assert.Equal(t, tc.expectedEventData, string(transformedEvent.Data()))

			if tc.expectedEventExtensions != "" {
				ext, err := json.Marshal(transformedEvent.Extensions())
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedEventExtensions, string(ext))
			}
		})
	}
}

func TestConcurrentStore(t *testing.T) {
	sharedStorage := storage.New()

	contextPipeline, err := newPipeline([]v1alpha1.Transform{
		{
			Operation: "store",
			Paths: []v1alpha1.Path{
				{
					Key:   "$id",
					Value: "id",
				},
			},
		},
	}, sharedStorage)
	assert.NoError(t, err)
	dataPipeline, err := newPipeline([]v1alpha1.Transform{
		{
			Operation: "add",
			Paths: []v1alpha1.Path{
				{
					Key:   "id",
					Value: "$id",
				},
			},
		},
	}, sharedStorage)
	assert.NoError(t, err)

	a := &adapter{
		DataPipeline:    dataPipeline,
		ContextPipeline: contextPipeline,
		logger:          logtesting.TestLogger(t),
	}

	transformationRequests := make(chan cloudevents.Event, 10)
	transformationResponses := make(chan cloudevents.Event, 10)

	// start 3 transformation workers
	for i := 0; i < 3; i++ {
		go func() {
			for event := range transformationRequests {
				result, err := a.applyTransformations(event)
				assert.NoError(t, err)
				transformationResponses <- *result
			}
		}()
	}

	// send 10 events
	for i := 0; i < 10; i++ {
		event := cloudevents.NewEvent(cloudevents.VersionV1)
		assert.NoError(t, event.SetData(cloudevents.ApplicationJSON, "{}"))
		event.SetID(fmt.Sprint(i))
		event.SetType("mock-event")
		event.SetSource("concurrency-test")
		assert.NoError(t, event.Validate())
		transformationRequests <- event
	}

	type data struct {
		ID string `json:"id"`
	}

	// receive transformed events
	counter := 0
	for result := range transformationResponses {
		var resultData data
		assert.NoError(t, result.DataAs(&resultData))

		if resultData.ID != result.ID() {
			assert.Failf(t, "transformation shared store race condition", "expected string: %v, actual string: %v", result.ID(), resultData.ID)
		}

		if counter++; counter == 10 {
			close(transformationResponses)
			close(transformationRequests)
		}
	}

	// ensure that the storage is being flushed properly
	assert.True(t, (len(a.ContextPipeline.Storage.ListEventIDs()) == 0) && (len(a.DataPipeline.Storage.ListEventIDs()) == 0))
}

func BenchmarkStorage(b *testing.B) {
	sharedStorage := storage.New()

	contextPipeline, _ := newPipeline([]v1alpha1.Transform{
		{
			Operation: "store",
			Paths: []v1alpha1.Path{
				{
					Key:   "$id",
					Value: "id",
				},
			},
		},
	}, sharedStorage)

	dataPipeline, _ := newPipeline([]v1alpha1.Transform{
		{
			Operation: "add",
			Paths: []v1alpha1.Path{
				{
					Key:   "id",
					Value: "$id",
				},
			},
		},
	}, sharedStorage)

	a := &adapter{
		DataPipeline:    dataPipeline,
		ContextPipeline: contextPipeline,

		logger: logtesting.TestLogger(b),
	}

	transformationRequests := make(chan cloudevents.Event)

	var wg sync.WaitGroup
	wg.Add(b.N)

	// start 10 transformation workers
	for i := 0; i < 10; i++ {
		go func() {
			for event := range transformationRequests {
				if _, err := a.applyTransformations(event); err != nil {
					b.Logf("transformation error: %v", err)
				}
				wg.Done()
			}
		}()
	}

	for i := 0; i < b.N; i++ {
		event := cloudevents.NewEvent(cloudevents.VersionV1)
		if err := event.SetData(cloudevents.ApplicationJSON, "{}"); err != nil {
			b.Logf("CE data error: %v", err)
			continue
		}
		event.SetType("mock-event")
		event.SetSource("benchmark-test")
		event.SetID(fmt.Sprint(i))

		transformationRequests <- event
	}
	wg.Wait()
}
