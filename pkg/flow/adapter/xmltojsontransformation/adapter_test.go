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

package xmltojsontransformation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cetest "github.com/cloudevents/sdk-go/v2/client/test"

	"knative.dev/eventing/pkg/adapter/v2"
	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	logtesting "knative.dev/pkg/logging/testing"

	"github.com/triggermesh/triggermesh/pkg/metrics"
	metricstesting "github.com/triggermesh/triggermesh/pkg/metrics/testing"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

const (
	tCloudEventID     = "ce-abcd-0123"
	tCloudEventType   = "ce.test.type"
	tCloudEventSource = "ce.test.source"

	tXML1        = `<note><to>Tove</to></note>`
	tJSONOutput1 = `{"note": {"to": "Tove"}}` + "\n"

	tFalseXML         = `"this is not xml"`
	tFalseXMLResponse = `{"Code":"request-validation","Description":"invalid XML","Details":null}`
)

func TestSink(t *testing.T) {
	testCases := map[string]struct {
		inEvent     cloudevents.Event
		expectEvent cloudevents.Event
	}{
		"sink ok": {
			inEvent:     newCloudEvent(t, tXML1, cloudevents.ApplicationXML),
			expectEvent: newCloudEvent(t, tJSONOutput1, cloudevents.ApplicationJSON),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			metricstesting.ResetMetrics(t)

			ceClient := adaptertest.NewTestClient()

			logger := logtesting.TestLogger(t)

			replier, err := targetce.New("test-xmltojson", logger)
			require.NoError(t, err)

			mt := &adapter.MetricTag{}

			a := &Adapter{
				sink:     "http://fake",
				replier:  replier,
				ceClient: ceClient,
				logger:   logger,

				mt: mt,
				sr: metrics.MustNewEventProcessingStatsReporter(mt),
			}

			ctx := context.Background()

			e, r := a.dispatch(ctx, tc.inEvent)
			assert.Nil(t, e)
			assert.Equal(t, cloudevents.ResultACK, r)

			events := ceClient.Sent()
			require.Equal(t, 1, len(events))
			assert.Equal(t, tc.expectEvent, events[0])
		})
	}
}

func TestReplier(t *testing.T) {
	testCases := map[string]struct {
		inEvent     cloudevents.Event
		expectEvent cloudevents.Event
	}{
		"transform ok": {
			inEvent:     newCloudEvent(t, tXML1, cloudevents.ApplicationXML),
			expectEvent: newCloudEvent(t, tJSONOutput1, cloudevents.ApplicationJSON),
		},
		"transform error": {
			inEvent:     newCloudEvent(t, tFalseXML, cloudevents.ApplicationXML),
			expectEvent: newCloudEvent(t, tFalseXMLResponse, cloudevents.ApplicationXML),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			metricstesting.ResetMetrics(t)

			ceClient, send, responses := cetest.NewMockResponderClient(t, 1)

			logger := logtesting.TestLogger(t)

			replier, err := targetce.New(tCloudEventSource, logger)
			require.NoError(t, err)

			mt := &adapter.MetricTag{}

			a := &Adapter{
				replier:  replier,
				ceClient: ceClient,
				logger:   logger,

				mt: mt,
				sr: metrics.MustNewEventProcessingStatsReporter(mt),
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
				assert.Equal(t, tCloudEventSource, event.Event.Source())
				assert.Equal(t, string(tc.expectEvent.DataEncoded), string(event.Event.DataEncoded))

			case <-time.After(2 * time.Second):
				assert.Fail(t, "expected cloud event response was not received")
			}

		})
	}
}

func newCloudEvent(t *testing.T, data, contentType string) cloudevents.Event {
	t.Helper()

	event := cloudevents.NewEvent()

	event.SetID(tCloudEventID)
	event.SetType(tCloudEventType)
	event.SetSource(tCloudEventSource)

	err := event.SetData(contentType, []byte(data))
	require.NoError(t, err)

	return event
}
