//go:build !noclibs

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

package xslttransformation

import (
	"context"
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cetest "github.com/cloudevents/sdk-go/v2/client/test"

	"knative.dev/eventing/pkg/adapter/v2"
	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	logtesting "knative.dev/pkg/logging/testing"

	xslt "github.com/wamuir/go-xslt"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	metricstesting "github.com/triggermesh/triggermesh/pkg/metrics/testing"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

const (
	tBridgeID               = "bridge-abdc-0123"
	tComponent              = "xslt-adapter"
	tCloudEventID           = "ce-abcd-0123"
	tCloudEventType         = "ce.test.type"
	tCloudEventResponseType = "ce.test.type.response"
	tCloudEventSource       = "ce.test.source"
	tSuccessAttribute       = "success"
	tErrorAttribute         = "error"

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
<output><item>A1</item><item>B2</item><item>C3</item></output>
`

	tAlternativeOutXML = `<?xml version="1.0"?>
<alt><item>A1</item><item>B2</item><item>C3</item></alt>
`
)

func TestNewTarget(t *testing.T) {
	testCases := map[string]struct {
		xslt        string
		expectPanic string
	}{
		"wrong XSLT": {
			xslt:        tFaultyXML,
			expectPanic: "XSLT validation error: failed to parse xsl",
		},

		"wrong configuration not providing XSLT": {
			xslt:        "",
			expectPanic: "if XSLT cannot be overriden by CloudEvent payloads, configured XSLT cannot be empty",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			metricstesting.UnregisterMetrics()

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

			env := &envAccessor{
				XSLT:              tc.xslt,
				AllowXSLTOverride: false,
			}

			ceClient, _, _ := cetest.NewMockResponderClient(t, 1)

			ctx := logtesting.TestContextWithLogger(t)

			_ = NewTarget(ctx, env, ceClient)
		})
	}
}

func TestXSLTTransformationEvents(t *testing.T) {
	testCases := map[string]struct {
		allowXSLTOverride bool
		xslt              string

		inEvent cloudevents.Event

		expectEvent    cloudevents.Event
		expectCategory string
	}{
		"transform ok": {
			allowXSLTOverride: false,
			xslt:              tXSLT,
			inEvent:           newCloudEvent(tXML, cloudevents.ApplicationXML),

			expectEvent:    newCloudEvent(tOutXML, cloudevents.ApplicationXML),
			expectCategory: tSuccessAttribute,
		},
		"transform setting override to true but not providing an extra XSLT, ok": {
			allowXSLTOverride: true,
			xslt:              tXSLT,
			inEvent:           newCloudEvent(tXML, cloudevents.ApplicationXML),

			expectEvent:    newCloudEvent(tOutXML, cloudevents.ApplicationXML),
			expectCategory: tSuccessAttribute,
		},
		"transform xslt at event, ok": {
			allowXSLTOverride: true,
			inEvent: newCloudEvent(
				createStructuredRequest(tXML, tXSLT),
				cloudevents.ApplicationJSON,
				cloudEventWithEventType(v1alpha1.EventTypeXSLTTransformation)),

			expectEvent:    newCloudEvent(tOutXML, cloudevents.ApplicationXML),
			expectCategory: tSuccessAttribute,
		},
		"transform xslt at event overrides default, ok": {
			allowXSLTOverride: true,
			xslt:              tXSLT,
			inEvent: newCloudEvent(
				createStructuredRequest(tXML, tAlternativeXSLT),
				cloudevents.ApplicationJSON,
				cloudEventWithEventType(v1alpha1.EventTypeXSLTTransformation)),

			expectEvent:    newCloudEvent(tAlternativeOutXML, cloudevents.ApplicationXML),
			expectCategory: tSuccessAttribute,
		},
		"malformed incoming event": {
			allowXSLTOverride: false,
			xslt:              tXSLT,
			inEvent:           newCloudEvent(tFaultyXML, cloudevents.ApplicationXML),

			expectEvent: newCloudEvent(
				createErrorResponse(targetce.ErrorCodeRequestValidation, "error processing XML with XSLT: xsl transformation failed"),
				cloudevents.ApplicationJSON),
			expectCategory: tErrorAttribute,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			metricstesting.ResetMetrics(t)

			logger := logtesting.TestLogger(t)

			ceClient, send, responses := cetest.NewMockResponderClient(t, 1)

			replier, err := targetce.New(tComponent, logger,
				targetce.ReplierWithStatefulHeaders(tBridgeID),
			)
			require.NoError(t, err)

			mt := &adapter.MetricTag{}

			a := &xsltTransformAdapter{
				xsltOverride: tc.allowXSLTOverride,

				replier:  replier,
				ceClient: ceClient,
				logger:   logger,

				mt: mt,
				sr: metrics.MustNewEventProcessingStatsReporter(mt),
			}

			if v := tc.xslt; v != "" {
				a.defaultXSLT, err = xslt.NewStylesheet([]byte(v))
				require.NoError(t, err)
				runtime.SetFinalizer(a.defaultXSLT, (*xslt.Stylesheet).Close)
			}

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

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

func TestXSLTTransformationToSink(t *testing.T) {
	testCases := map[string]struct {
		xslt          string
		inEvent       cloudevents.Event
		expectedEvent cloudevents.Event
	}{
		"transform ok": {
			xslt:          tXSLT,
			inEvent:       newCloudEvent(tXML, cloudevents.ApplicationXML),
			expectedEvent: newCloudEvent(tOutXML, cloudevents.ApplicationXML, cloudEventWithEventType(tCloudEventResponseType)),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ceClient := adaptertest.NewTestClient()
			style, err := xslt.NewStylesheet([]byte(tc.xslt))
			assert.NoError(t, err)

			a := &xsltTransformAdapter{
				logger: logtesting.TestLogger(t),

				ceClient:     ceClient,
				xsltOverride: false,
				defaultXSLT:  style,
				sink:         "http://localhost:8080",
			}

			ctx := context.Background()

			e, r := a.dispatch(ctx, tc.inEvent)
			assert.Nil(t, e)
			assert.Equal(t, cloudevents.ResultACK, r)

			events := ceClient.Sent()
			require.Equal(t, 1, len(events))
			t.Log(string(events[0].Data()))
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

func createStructuredRequest(xml, xslt string) string {
	sr := XSLTTransformationStructuredRequest{
		XML:  xml,
		XSLT: xslt,
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
