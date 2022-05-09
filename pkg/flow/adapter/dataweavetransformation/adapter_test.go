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

package dataweavetransformation

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cetest "github.com/cloudevents/sdk-go/v2/client/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
	"knative.dev/eventing/pkg/adapter/v2"
	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	logtesting "knative.dev/pkg/logging/testing"
)

const (
	tBridgeID               = "bride-abdc-0123"
	tCloudEventID           = "ce-abcd-0123"
	tCloudEventType         = "ce.test.type"
	tCloudEventResponseType = "ce.test.type.response"
	tCloudEventSource       = "ce.test.source"
	tSuccessAttribute       = "success"
	tErrorAttribute         = "error"

	tDwSpell = `%dw 2.0
output application/json
---
items: payload.books map (item, index) -> {
		book: item mapObject (value, key) -> {
		(upper(key)): value
		}
	}
`

	tDwSpellAlternative = `%dw 2.0
output application/json
---
{
		address1: payload.order.buyer.address,
		city: payload.order.buyer.city,
		country: payload.order.buyer.nationality,
		email: payload.order.buyer.email,
		name: payload.order.buyer.name,
		postalCode: payload.order.buyer.postCode,
		stateOrProvince: payload.order.buyer.state
}
`
	tJSON = `
	{
		"books": [{
				"-category": "cooking",
				"title": "Everyday Italian",
				"author": "Giada De Laurentiis",
				"year": "2005",
				"price": "30.00"
			},
			{
				"-category": "children",
				"title": "Harry Potter",
				"author": "J K. Rowling",
				"year": "2005",
				"price": "29.99"
			},
			{
				"-category": "web",
				"title": "XQuery Kick Start",
				"author": [
					"James McGovern",
					"Per Bothner",
					"Kurt Cagle",
					"James Linn",
					"Vaidyanathan Nagarajan"
				],
				"year": "2003",
				"price": "49.99"
			},
			{
				"-category": "web",
				"-cover": "paperback",
				"title": "Learning XML",
				"author": "Erik T. Ray",
				"year": "2003",
				"price": "39.95"
			}
		]
	}
`

	tOutJSON = `
{
  "items": [
    {
      "book": {
        "-CATEGORY": "cooking",
        "TITLE": "Everyday Italian",
        "AUTHOR": "Giada De Laurentiis",
        "YEAR": "2005",
        "PRICE": "30.00"
      }
    },
    {
      "book": {
        "-CATEGORY": "children",
        "TITLE": "Harry Potter",
        "AUTHOR": "J K. Rowling",
        "YEAR": "2005",
        "PRICE": "29.99"
      }
    },
    {
      "book": {
        "-CATEGORY": "web",
        "TITLE": "XQuery Kick Start",
        "AUTHOR": [
          "James McGovern",
          "Per Bothner",
          "Kurt Cagle",
          "James Linn",
          "Vaidyanathan Nagarajan"
        ],
        "YEAR": "2003",
        "PRICE": "49.99"
      }
    },
    {
      "book": {
        "-CATEGORY": "web",
        "-COVER": "paperback",
        "TITLE": "Learning XML",
        "AUTHOR": "Erik T. Ray",
        "YEAR": "2003",
        "PRICE": "39.95"
      }
    }
  ]
}`

	tAlternativeXML = `
<order>       
	<product>
	<price>5</price>           
	<model>MuleSoft Connect 2016</model>
	</product>                            
	<item_amount>3</item_amount>
	<payment>             
	<payment-type>credit-card</payment-type>
	<currency>USD</currency>
	<installments>1</installments>
	</payment>                    
	<buyer>                        
	<email>mike@hotmail.com</email>
	<name>Michael</name>
	<address>Koala Boulevard 314</address>
	<city>San Diego</city>
	<state>CA</state>      
	<postCode>1345</postCode>         
	<nationality>USA</nationality>
	</buyer>                 
	<shop>main branch</shop>
	<salesperson>Mathew Chow</salesperson>
</order>
`

	tOutAlternativeJSON = `
{
  "address1": "Koala Boulevard 314",
  "city": "San Diego",
  "country": "USA",
  "email": "mike@hotmail.com",
  "name": "Michael",
  "postalCode": "1345",
  "stateOrProvince": "CA"
}`

	tInvalidJSON = "I am not valid JSON"
)

func TestDataWeaveTransformationEvents(t *testing.T) {
	testCases := map[string]struct {
		allowDwSpellOverride bool
		inputContentType     string
		outputContentType    string
		dwSpell              string

		inEvent cloudevents.Event

		expectPanic    string
		expectEvent    cloudevents.Event
		expectCategory string
	}{
		"transform ok": {
			inputContentType:  "application/json",
			outputContentType: "application/json",
			dwSpell:           tDwSpell,
			inEvent:           newCloudEvent(tJSON, cloudevents.ApplicationJSON),

			expectEvent:    newCloudEvent(tOutJSON, cloudevents.ApplicationJSON),
			expectCategory: tSuccessAttribute,
		},
		"transform setting override to true but not providing an extra spell, ok": {
			allowDwSpellOverride: true,
			inputContentType:     "application/json",
			outputContentType:    "application/json",
			dwSpell:              tDwSpell,
			inEvent:              newCloudEvent(tJSON, cloudevents.ApplicationJSON),

			expectEvent:    newCloudEvent(tOutJSON, cloudevents.ApplicationJSON),
			expectCategory: tSuccessAttribute,
		},
		"transform spell at event, ok": {
			allowDwSpellOverride: true,
			inEvent: newCloudEvent(
				createStructuredRequest(tDwSpell, tJSON, "application/json", "application/json"),
				cloudevents.ApplicationJSON,
				cloudEventWithEventType(v1alpha1.EventTypeDataWeaveTransformation)),

			expectEvent:    newCloudEvent(tOutJSON, cloudevents.ApplicationJSON),
			expectCategory: tSuccessAttribute,
		},
		"transform xml ok": {
			inputContentType:  "application/xml",
			outputContentType: "application/json",
			dwSpell:           tDwSpellAlternative,
			inEvent:           newCloudEvent(tAlternativeXML, cloudevents.ApplicationXML),

			expectEvent:    newCloudEvent(tOutAlternativeJSON, cloudevents.ApplicationJSON),
			expectCategory: tSuccessAttribute,
		},
		"malformed incoming event": {
			inputContentType:  "application/json",
			outputContentType: "application/json",
			dwSpell:           tDwSpell,
			inEvent:           newCloudEvent(tInvalidJSON, cloudevents.ApplicationJSON),

			expectEvent: newCloudEvent(
				createErrorResponse(targetce.ErrorCodeRequestValidation, "invalid Json"),
				cloudevents.ApplicationJSON),
			expectCategory: tErrorAttribute,
		},
		"unexpected incoming event": {
			inputContentType:  "application/json",
			outputContentType: "application/json",
			dwSpell:           tDwSpell,
			inEvent:           newCloudEvent(tInvalidJSON, "application/jsonn"),

			expectEvent: newCloudEvent(
				createErrorResponse(targetce.ErrorCodeRequestValidation, "unexpected type for the incoming event"),
				cloudevents.ApplicationJSON),
			expectCategory: tErrorAttribute,
		},
		"transform spell at event, malformed incoming event": {
			allowDwSpellOverride: true,
			inEvent: newCloudEvent(
				createStructuredRequest(tDwSpell, tInvalidJSON, "application/json", "application/json"),
				cloudevents.ApplicationJSON,
				cloudEventWithEventType(v1alpha1.EventTypeDataWeaveTransformation)),

			expectEvent: newCloudEvent(
				createErrorResponse(targetce.ErrorCodeRequestValidation, "invalid Json"),
				cloudevents.ApplicationJSON),
			expectCategory: tErrorAttribute,
		},
		"transform spell at event, missing inputData": {
			allowDwSpellOverride: true,
			inputContentType:     "application/json",
			outputContentType:    "application/json",
			dwSpell:              tDwSpell,
			inEvent: newCloudEvent(
				createStructuredRequest(tDwSpell, "", "application/json", "application/json"),
				cloudevents.ApplicationJSON,
				cloudEventWithEventType(v1alpha1.EventTypeDataWeaveTransformation)),

			expectEvent: newCloudEvent(
				createErrorResponse(targetce.ErrorCodeRequestValidation, "inputData not found"),
				cloudevents.ApplicationJSON),
			expectCategory: tErrorAttribute,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			defer func() {
				r := recover()
				switch {
				case r == nil:
					assert.Empty(t, tc.expectPanic, "Expected panic did not occur")
				case tc.expectPanic == "":
					assert.Fail(t, "Unexpected panic", r)
				default:
					assert.Contains(t, r, tc.expectPanic)
				}
			}()

			ctx := context.Background()
			logtesting.TestContextWithLogger(t)

			env := &envAccessor{
				EnvConfig: adapter.EnvConfig{
					Component: tCloudEventSource,
				},
				AllowDwSpellOverride: tc.allowDwSpellOverride,
				DwSpell:              tc.dwSpell,
				InputContentType:     tc.inputContentType,
				OutputContentType:    tc.outputContentType,
				BridgeIdentifier:     tBridgeID,
			}

			ceClient, send, responses := cetest.NewMockResponderClient(t, 1)

			a := NewTarget(ctx, env, ceClient)

			go func() {
				if err := a.Start(ctx); err != nil {
					assert.FailNow(t, "could not start test adapter")
				}
			}()

			send <- tc.inEvent

			select {
			case event := <-responses:
				assert.Equal(t, tCloudEventSource, event.Event.Source())
				assert.Equal(t, string(tc.expectEvent.DataEncoded), string(event.Event.DataEncoded))

			case <-time.After(15 * time.Second):
				assert.Fail(t, "expected cloud event response was not received")
			}

		})
	}
}

func TestDataWeaveTransformationToSink(t *testing.T) {
	testCases := map[string]struct {
		allowDwSpellOverride bool
		incomingContentType  string
		outputContentType    string
		dwSpell              string
		inEvent              cloudevents.Event
		expectedEvent        cloudevents.Event
	}{
		"transform ok": {
			incomingContentType: "application/json",
			outputContentType:   "application/json",
			dwSpell:             tDwSpell,
			inEvent:             newCloudEvent(tJSON, cloudevents.ApplicationJSON),
			expectedEvent:       newCloudEvent(tOutJSON, cloudevents.ApplicationJSON, cloudEventWithEventType(tCloudEventResponseType)),
		},
		"transform xml ok": {
			incomingContentType: "application/xml",
			outputContentType:   "application/json",
			dwSpell:             tDwSpellAlternative,
			inEvent:             newCloudEvent(tAlternativeXML, cloudevents.ApplicationXML),
			expectedEvent:       newCloudEvent(tOutAlternativeJSON, cloudevents.ApplicationJSON, cloudEventWithEventType(tCloudEventResponseType)),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ceClient := adaptertest.NewTestClient()
			ctx := context.Background()

			mt := &adapter.MetricTag{}

			a := &dataweaveTransformAdapter{
				logger: logtesting.TestLogger(t),

				ceClient:                 ceClient,
				defaultInputContentType:  &tc.incomingContentType,
				defaultOutputContentType: &tc.outputContentType,
				defaultSpell:             &tc.dwSpell,
				sink:                     "http://localhost:8080",
				mt:                       mt,
				sr:                       metrics.MustNewEventProcessingStatsReporter(mt),
			}

			e, r := a.dispatch(ctx, tc.inEvent)
			assert.Nil(t, e)
			assert.Equal(t, cloudevents.ResultACK, r)

			events := ceClient.Sent()
			require.Equal(t, 1, len(events))
			assert.Equal(t, tc.expectedEvent, events[0])
		})
	}
}

type cloudEventOptions func(*cloudevents.Event)

func newCloudEvent(data, contentType string, opts ...cloudEventOptions) cloudevents.Event {
	event := cloudevents.NewEvent()

	if err := event.SetData(contentType, []byte(data)); err != nil {
		// not expected
		panic(err)
	}

	event.SetID(tCloudEventID)
	event.SetType(tCloudEventType)
	event.SetSource(tCloudEventSource)

	for _, o := range opts {
		o(&event)
	}

	return event
}

func cloudEventWithEventType(t string) cloudEventOptions {
	return func(ce *cloudevents.Event) {
		ce.SetType(t)
	}
}

func createStructuredRequest(spell, inputData, inputContentType, outputContentType string) string {
	sr := DataWeaveTransformationStructuredRequest{
		Spell:             spell,
		InputData:         inputData,
		InputContentType:  inputContentType,
		OutputContentType: outputContentType,
	}

	b, err := json.Marshal(sr)
	if err != nil {
		// not expected
		panic(err)
	}

	return string(b)
}

func createErrorResponse(code, description string) string {
	ee := targetce.EventError{
		Code:        code,
		Description: description,
	}

	b, err := json.Marshal(ee)
	if err != nil {
		// not expected
		panic(err)
	}

	return string(b)
}
