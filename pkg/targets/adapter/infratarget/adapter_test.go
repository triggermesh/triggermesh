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

package infratarget

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	cetest "github.com/cloudevents/sdk-go/v2/client/test"
	ceevent "github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	jsvm "github.com/triggermesh/triggermesh/pkg/targets/adapter/infratarget/vm/javascript"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	logtesting "knative.dev/pkg/logging/testing"
)

const (
	tInputData   = `{"key1":"value1","key2":"value2", "key3": true, "key4":"hello\nworld"}`
	tID          = "abc123"
	tContentType = "application/json"
	tCEType      = "test.type"
	tCESource    = "test.source"
	tStateBridge = "bridge-triggermesh-test"
	tStateID     = "b00001"
	tStateStep   = "step1.1"

	tContentTypeXML = "application/xml"
	tInputDataXML   = `<A><B><B1>value B1</B1><B2 attributeB2="value-attr-B2">value B2</B2></B><C/><D attributeC="1"/></A>`
)

func TestInfraRequests(t *testing.T) {

	testCases := map[string]struct {
		// adapter config
		userScript         string
		inputCE            *ceevent.Event
		stateHeadersPolicy string
		stateBridge        string
		typeLoopProtection bool

		expectNil        bool
		expectResult     string
		expectData       string
		expectType       string
		expectSource     string
		expectExtensions map[string]interface{}
		expectLogs       []logEntry
	}{
		"New event, nested response ": {
			userScript: `
			function handle(input) {
				nevent = {
					data: {
						"hello": "world",
						"nest": {"nested1": [1,2,3],	"nested2": "nestedvalue"}},
					"type": "test.response.type",
				}

				return nevent
			}
			`,
			inputCE: createJSONEvent(t),

			expectNil:    false,
			expectData:   `{"hello":"world","nest":{"nested1":[1,2,3],"nested2":"nestedvalue"}}`,
			expectType:   "test.response.type",
			expectSource: tCESource,
		},

		"New event, use input variable": {
			userScript: `
			function handle(input) {
				nevent = {
					data: {
						"hello": "world",
						"works": input.data.key3,
						"nest": {"nested1": [1,2,3],	"nested2": input.data.key1}},
					"type": "test.response.type",
				}

				return nevent
			}
			`,
			inputCE: createJSONEvent(t),

			expectNil:    false,
			expectData:   `{"hello":"world","nest":{"nested1":[1,2,3],"nested2":"value1"},"works":true}`,
			expectType:   "test.response.type",
			expectSource: tCESource,
		},

		"Nil response": {
			userScript: `
			function handle(input) {
				if (input.data.key3 == true) {
					return;
				}

				nevent = {
					data: {"hello": "world"},
					"type": "test.response.type",
				}

				return nevent
			}
			`,
			inputCE: createJSONEvent(t),

			expectNil: true,
		},

		"New event, missing event type": {
			userScript: `
			function handle(input) {
				nevent = {
					data: {"hello": "world"},
				}

				return nevent
			}
			`,
			inputCE: createJSONEvent(t),

			expectData:   `{"hello":"world"}`,
			expectSource: tCESource,
		},

		"New event, overwrite source": {
			userScript: `
			function handle(input) {
				nevent = {
					data: {"hello": "world"},
					"source": "new.source",
				}

				return nevent
			}
			`,
			inputCE: createJSONEvent(t),

			expectData:   `{"hello":"world"}`,
			expectSource: "new.source",
		},

		"Input event, add elements": {
			userScript: `
			function handle(input) {
				input.data.extra = "extra-value"
				input.data.nested = {"k1":"v1","k2":"v2"}
				input.type = "test.response.type"

				return input
			}
			`,
			inputCE: createJSONEvent(t),

			expectData:   `{"extra":"extra-value","key1":"value1","key2":"value2","key3":true,"key4":"hello\nworld","nested":{"k1":"v1","k2":"v2"}}`,
			expectType:   "test.response.type",
			expectSource: tCESource,
		},

		"Input event, with extension": {
			userScript: `
			function handle(input) {
				input.type = "test.response.type"

				return input
			}
			`,
			inputCE: createJSONEvent(t, withExtension("stateid", "st0001")),

			expectData:       `{"key1":"value1","key2":"value2","key3":true,"key4":"hello\nworld"}`,
			expectType:       "test.response.type",
			expectSource:     tCESource,
			expectExtensions: map[string]interface{}{"stateid": "st0001"},
		},

		"Input event, add header": {
			userScript: `
			function handle(input) {
				input.newheader = "value-new-header"
				input.type = "test.response.type"

				return input
			}
			`,
			inputCE: createJSONEvent(t),

			expectData:       `{"key1":"value1","key2":"value2","key3":true,"key4":"hello\nworld"}`,
			expectType:       "test.response.type",
			expectSource:     tCESource,
			expectExtensions: map[string]interface{}{"newheader": "value-new-header"},
		},

		"Input event, delete extension": {
			userScript: `
			function handle(input) {
				delete input["stateid"]
				input.type = "test.response.type"

				return input
			}
			`,
			inputCE: createJSONEvent(t, withExtension("stateid", "st0001")),

			expectData:   `{"key1":"value1","key2":"value2","key3":true,"key4":"hello\nworld"}`,
			expectType:   "test.response.type",
			expectSource: tCESource,
		},

		"Script syntax error": {
			userScript: `
			function handle(input) {
				a { " ;
			}
			`,
			inputCE: createJSONEvent(t),

			expectNil:    true,
			expectResult: "Unexpected token",
		},

		"Script runtime error": {
			userScript: `
			function handle(input) {
				throw "testing errors while running";
			}
			`,
			inputCE: createJSONEvent(t),

			expectNil:    true,
			expectResult: "testing errors while running",
		},

		"Script to adapter logger": {
			userScript: `
			function handle(input) {
				a = 42
				log("answer to the ultimate question: " + a)
			}
			`,
			inputCE: createJSONEvent(t),

			expectNil:  true,
			expectLogs: []logEntry{{message: "answer to the ultimate question: 42", level: zapcore.InfoLevel}},
		},

		"Timeout": {
			userScript: `
			function handle(input) {
				a = 0
				for (i = 0; i < 10000000; i++){
					a = a + 1;
				}

				log("a iterations: " + a)
			}
			`,
			inputCE: createJSONEvent(t),

			expectNil:    true,
			expectResult: "VM execution timed out",
		},

		"Ensure state headers": {
			userScript: `
			function handle(input) {
				nevent = {
					data: {"hello": "world"},
					"source": "new.source",
					"statefulid": "123456"
				}

				return nevent
			}
			`,
			inputCE:            createJSONEvent(t),
			stateHeadersPolicy: "ensure",
			stateBridge:        tStateBridge,

			expectData:       `{"hello":"world"}`,
			expectSource:     "new.source",
			expectExtensions: map[string]interface{}{"statefulbridge": tStateBridge, "statefulid": "123456"},
		},

		"Propagate state headers": {
			userScript: `
			function handle(input) {
				nevent = {
					data: {"hello": "world"},
					"source": "new.source",
				}

				return nevent
			}
			`,
			inputCE: createJSONEvent(t,
				withExtension("statefulbridge", tStateBridge),
				withExtension("statefulid", tStateID),
				withExtension("statestep", tStateStep)),
			stateHeadersPolicy: "propagate",

			expectData:   `{"hello":"world"}`,
			expectSource: "new.source",
			expectExtensions: map[string]interface{}{
				"statefulbridge": tStateBridge,
				"statefulid":     tStateID,
				"statestep":      tStateStep,
			},
		},

		"Overwrite step state header": {
			userScript: `
			function handle(input) {
				nevent = {
					data: {"hello": "world"},
					"source": "new.source",
				}

				if (input.statestep == "` + tStateStep + `") {
					nevent.statestep = "new.step";
				}

				return nevent
			}
			`,
			inputCE: createJSONEvent(t,
				withExtension("statefulbridge", tStateBridge),
				withExtension("statefulid", tStateID),
				withExtension("statestep", tStateStep)),
			stateHeadersPolicy: "propagate",

			expectData:   `{"hello":"world"}`,
			expectSource: "new.source",
			expectExtensions: map[string]interface{}{
				"statefulbridge": tStateBridge,
				"statefulid":     tStateID,
				"statestep":      "new.step",
			},
		},

		"No headers overriden at nil return": {
			userScript: `
			function handle(input) {
				return
			}
			`,
			inputCE: createJSONEvent(t,
				withExtension("statefulbridge", tStateBridge),
				withExtension("statefulid", tStateID),
				withExtension("statestep", tStateStep)),
			stateHeadersPolicy: "propagate",

			expectNil: true,
		},

		"Type loop protection": {
			userScript: `
			function handle(input) {
				return input
			}
			`,
			inputCE: createJSONEvent(t,
				withExtension("statefulbridge", tStateBridge),
				withExtension("statefulid", tStateID),
				withExtension("statestep", tStateStep)),
			typeLoopProtection: true,

			expectNil:    true,
			expectResult: `incoming and outgoing CloudEvents have the same type "test.type"`,
		},

		"Need to scape %": {
			userScript: `
			function handle(input) {

				output = {}
				output.type = "test.response.type"
				output.source = "test.response.source"
				output.data = {"escape-this": "% %s %n %%"}

				return output
			}
			`,
			inputCE: createJSONEvent(t),

			expectData:   `{"escape-this":"% %s %n %%"}`,
			expectSource: "test.response.source",
			expectType:   "test.response.type",
		},

		"Report exact line when failing": {
			userScript: `
			function handle(input) {
				log("syntax error")a;
			}
			`,
			inputCE: createJSONEvent(t),

			expectResult: `Line 3`,
			expectNil:    true,
		},

		"Error if handle is not implemented": {
			userScript: `
			function handleme(input) {
				log("I have a wrong name");
			}
			`,
			inputCE: createJSONEvent(t),

			expectResult: `script does not implement handle`,
			expectNil:    true,
		},

		"Error if handle function has no parameters": {
			userScript: `
			function handle() {
				log("handle needs 1 param");
			}
			`,
			inputCE: createJSONEvent(t),

			expectResult: `handle(input) function accepts exactly one parameter`,
			expectNil:    true,
		},

		"XML event": {
			userScript: `
			function handle(input) {
				log(input)
				nevent = {
					data: {
						"B1Key": input.data.A.B.B1,
						"B2Key": input.data.A.B.B2,
						"CKey": input.data.A.C,
						"DKey": input.data.A.D,
					},
					"type": "test.response.type",
				}

				return nevent
			}
			`,
			inputCE:      createXMLEvent(t),
			expectNil:    false,
			expectData:   `{"B1Key":"value B1","B2Key":{"#text":"value B2","-attributeB2":"value-attr-B2"},"CKey":"","DKey":{"-attributeC":"1"}}`,
			expectType:   "test.response.type",
			expectSource: tCESource,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			ceClient, send, responses := cetest.NewMockResponderClient(t, 1)

			var logger *zap.SugaredLogger
			var logs *observer.ObservedLogs

			if len(tc.expectLogs) == 0 {
				logger = logtesting.TestLogger(t)
			} else {
				// observer wont trace log at stdout, hence we only setup
				// this one when checking logs
				var core zapcore.Core
				core, logs = observer.New(zap.InfoLevel)
				logger = zap.New(core).Sugar()
			}

			adapter := &infraAdapter{
				ceClient:           ceClient,
				logger:             logger,
				typeLoopProtection: tc.typeLoopProtection,
			}

			if tc.userScript != "" {
				adapter.vm = jsvm.New(tc.userScript, time.Second, logger.Named("vm"))
			}

			switch tc.stateHeadersPolicy {
			case "ensure":
				adapter.preProcessHeaders = ensureStateHeaders(tc.stateBridge)
				fallthrough
			case "propagate":
				adapter.postProcessHeaders = propagateStateHeaders
			}

			go func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				if err := adapter.Start(ctx); err != nil {
					assert.FailNow(t, err.Error(), "could not start test adapter")
				}
			}()

			send <- *tc.inputCE

			select {
			case event := <-responses:

				// nested loop, we expect 1 to 3 entries for each
				for _, el := range tc.expectLogs {
					match := false
					for _, l := range logs.All() {
						if l.Level == el.level && l.Message == el.message {
							match = true
							break
						}
					}
					assert.Truef(t, match, "expected log entry was not found: [%s] %s", el.level, el.message)
				}

				if tc.expectNil {
					assert.Nil(t, event.Event.Data(), "expected nil response but got cloud event data")
					assert.Contains(t, event.Result.Error(), tc.expectResult, "unexpected result error data")
					return
				}

				if event.Event.Data() == nil {
					assert.Fail(t, "reponse cloud event was nil")
				}

				assert.Equal(t, tc.expectType, event.Event.Type(), "response type does not match expectations")
				assert.Equal(t, tc.expectSource, event.Event.Source(), "response type does not match expectations")
				assert.Equal(t, tc.expectData, string(event.Event.Data()), "response data does not match expectations")

				extensions := event.Event.Extensions()
				assert.Equal(t, tc.expectExtensions, extensions, "response extensions did not match the expect value")

			case <-time.After(3 * time.Second):
				assert.Fail(t, "expected cloud event response was not received")
			}
		})
	}
}

type eventOpts func(*ceevent.Event)

func createJSONEvent(t *testing.T, opts ...eventOpts) *ceevent.Event {
	data := make(map[string]interface{})
	if err := json.Unmarshal([]byte(tInputData), &data); err != nil {
		assert.Fail(t, err.Error(), "could not read input data for base CloudEvent: %v", data)
	}
	return createBaseEvent(t, tContentType, data, opts...)
}

func createXMLEvent(t *testing.T, opts ...eventOpts) *ceevent.Event {
	return createBaseEvent(t, tContentTypeXML, []byte(tInputDataXML), opts...)
}

func withExtension(key string, value interface{}) eventOpts {
	return func(event *ceevent.Event) {
		event.SetExtension(key, value)
	}
}

type logEntry struct {
	message string
	level   zapcore.Level
}

func createBaseEvent(t *testing.T, contentType string, data interface{}, opts ...eventOpts) *ceevent.Event {
	event := ceevent.New()
	if err := event.SetData(contentType, data); err != nil {
		assert.Fail(t, err.Error(), "failed to set data for the event: %+v", data)
	}

	event.SetID(tID)
	event.SetType(tCEType)
	event.SetSource(tCESource)

	for _, f := range opts {
		f(&event)
	}

	return &event
}
