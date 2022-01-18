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
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
)

var availableTransformations = []v1alpha1.Transform{
	{Operation: "add"},
	{Operation: "store"},
	{Operation: "shift"},
	{Operation: "delete"},
}

func TestNewHandler(t *testing.T) {
	_, err := NewHandler(availableTransformations, availableTransformations)
	assert.NoError(t, err)
}

func TestStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	pipeline, err := NewHandler(availableTransformations, availableTransformations)
	assert.NoError(t, err)

	errChan := make(chan error)

	go func() {
		defer close(errChan)
		errChan <- pipeline.Start(ctx, "")
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

func newEvent() cloudevents.Event {
	emptyV1Event := cloudevents.NewEvent(cloudevents.VersionV1)
	emptyV1Event.SetID("123")
	emptyV1Event.SetSource("test")
	emptyV1Event.SetType("test")
	return emptyV1Event
}

func TestReceiveAndTransform(t *testing.T) {
	testCases := []struct {
		name              string
		originalEvent     cloudevents.Event
		expectedEventData string
		data              []v1alpha1.Transform
	}{
		{
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
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pipeline, err := NewHandler([]v1alpha1.Transform{}, tc.data)
			assert.NoError(t, err)

			transformedEvent, err := pipeline.applyTransformations(tc.originalEvent)
			assert.NoError(t, err)

			assert.Equal(t, tc.expectedEventData, string(transformedEvent.Data()))
		})
	}
}
