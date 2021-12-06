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

package xslttransform

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cetest "github.com/cloudevents/sdk-go/v2/client/test"
	"github.com/stretchr/testify/assert"

	"knative.dev/eventing/pkg/adapter/v2"
	logtesting "knative.dev/pkg/logging/testing"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

const (
	tBridgeID         = "bride-abdc-0123"
	tComponent        = "xslt-adapter"
	tCloudEventID     = "ce-abcd-0123"
	tCloudEventType   = "ce.test.type"
	tCloudEventSource = "ce.test.source"
	tSuccessAttribute = "success"
	tErrorAttribute   = "error"

	tXSLT = `
<xsl:stylesheet version="1.0"	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="tests">
    <output>
      <xsl:apply-templates select="test">
        <xsl:sort select="data/el1"/>
        <xsl:sort select="data/el2"/>
      </xsl:apply-templates>
    </output>
  </xsl:template>

  <xsl:template match="test">
    <item>
      <xsl:value-of select="data/el1"/>
      <xsl:value-of select="data/el2"/>
    </item>
  </xsl:template>
</xsl:stylesheet>
`

	tAlternativeXSLT = `
<xsl:stylesheet version="1.0"	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="tests">
    <alt>
      <xsl:apply-templates select="test">
        <xsl:sort select="data/el1"/>
        <xsl:sort select="data/el2"/>
      </xsl:apply-templates>
    </alt>
  </xsl:template>

  <xsl:template match="test">
    <item>
      <xsl:value-of select="data/el1"/>
      <xsl:value-of select="data/el2"/>
    </item>
  </xsl:template>
</xsl:stylesheet>
`

	tFaultyXML = `this is not XSLT`

	tXML = `
<tests>
  <test>
    <data>
      <el1>A</el1>
      <el2>1</el2>
    </data>
  </test>
  <test>
    <data>
			<el1>B</el1>
			<el2>2</el2>
    </data>
  </test>
  <test>
    <data>
			<el1>C</el1>
			<el2>3</el2>
    </data>
  </test>
</tests>
`

	tOutXML = `<?xml version="1.0"?>
<output>
  <item>A1</item>
  <item>B2</item>
  <item>C3</item>
</output>
`

	tAlternativeOutXML = `<?xml version="1.0"?>
<alt>
  <item>A1</item>
  <item>B2</item>
  <item>C3</item>
</alt>
`
)

func TestXsltTransformEvents(t *testing.T) {
	testCases := map[string]struct {
		allowXsltOverride bool
		xslt              string

		inEvent cloudevents.Event

		expectPanic    string
		expectEvent    cloudevents.Event
		expectCategory string
	}{
		"transform ok": {
			allowXsltOverride: false,
			xslt:              tXSLT,
			inEvent:           newCloudEvent(tXML, cloudevents.ApplicationXML),

			expectEvent:    newCloudEvent(tOutXML, cloudevents.ApplicationXML),
			expectCategory: tSuccessAttribute,
		},
		"transform setting override to true but not providing an extra XSLT, ok": {
			allowXsltOverride: true,
			xslt:              tXSLT,
			inEvent:           newCloudEvent(tXML, cloudevents.ApplicationXML),

			expectEvent:    newCloudEvent(tOutXML, cloudevents.ApplicationXML),
			expectCategory: tSuccessAttribute,
		},
		"transform xslt at event, ok": {
			allowXsltOverride: true,
			inEvent: newCloudEvent(
				createStructuredRequest(tXML, tXSLT),
				cloudevents.ApplicationJSON,
				cloudEventWithEventType(v1alpha1.EventTypeXsltTransform)),

			expectEvent:    newCloudEvent(tOutXML, cloudevents.ApplicationXML),
			expectCategory: tSuccessAttribute,
		},

		"transform xslt at event overrides default, ok": {
			allowXsltOverride: true,
			xslt:              tXSLT,
			inEvent: newCloudEvent(
				createStructuredRequest(tXML, tAlternativeXSLT),
				cloudevents.ApplicationJSON,
				cloudEventWithEventType(v1alpha1.EventTypeXsltTransform)),

			expectEvent:    newCloudEvent(tAlternativeOutXML, cloudevents.ApplicationXML),
			expectCategory: tSuccessAttribute,
		},

		"wrong XSLT": {
			allowXsltOverride: false,
			xslt:              tFaultyXML,

			expectPanic: "Non valid XSLT document",
		},

		"wrong configuration not providing XSLT": {
			allowXsltOverride: false,
			xslt:              "",

			expectPanic: "if XSLT cannot be overriden by CloudEvent payloads, configured XSLT cannot be empty",
		},

		"malformed incoming event": {
			allowXsltOverride: false,
			xslt:              tXSLT,
			inEvent:           newCloudEvent(tFaultyXML, cloudevents.ApplicationXML),

			expectEvent: newCloudEvent(
				createErrorResponse(targetce.ErrorCodeRequestParsing, "failed to parse xml input"),
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
					Component: tComponent,
				},
				Xslt:              tc.xslt,
				AllowXsltOverride: tc.allowXsltOverride,
				BridgeIdentifier:  tBridgeID,
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
				assert.Equal(t, tComponent, event.Event.Source())
				assert.Equal(t, tc.expectCategory, event.Event.Extensions()["category"])
				assert.Equal(t, tBridgeID, event.Event.Extensions()["statefulbridge"])
				assert.NotEmpty(t, event.Event.Extensions()["statefulbridge"])

				assert.Equal(t, string(tc.expectEvent.DataEncoded), string(event.Event.DataEncoded))

			case <-time.After(15 * time.Second):
				assert.Fail(t, "expected cloud event response was not received")
			}

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

func createStructuredRequest(xml, xslt string) string {
	sr := XsltTransformStructuredRequest{
		XML:  xml,
		XSLT: xslt,
	}

	b, err := json.Marshal(sr)
	if err != nil {
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
		panic(err)
	}

	return string(b)
}
