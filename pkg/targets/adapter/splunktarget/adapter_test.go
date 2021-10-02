/*
Copyright 2020 TriggerMesh Inc.

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

package splunktarget

import (
	"context"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	logtesting "knative.dev/pkg/logging/testing"

	"github.com/ZachtimusPrime/Go-Splunk-HTTP/splunk/v2"
)

const tDefaultIndex = "fake-index"

type mockedSplunkClient struct {
	err           error
	inputRecorder []*splunk.Event
}

var _ SplunkClient = (*mockedSplunkClient)(nil)

func (c *mockedSplunkClient) NewEventWithTime(t time.Time, event interface{}, source, sourcetype, index string) *splunk.Event {
	return (&splunk.Client{}).NewEventWithTime(t, event, source, sourcetype, index)
}

func (c *mockedSplunkClient) LogEvent(in *splunk.Event) error {
	c.inputRecorder = append(c.inputRecorder, in)

	if c.err != nil {
		return c.err
	}
	return nil
}

// TestReceive verifies that a received event gets forwarded to Splunk.
func TestReceive(t *testing.T) {
	testCases := map[string]struct {
		client       *mockedSplunkClient
		expectEvent  *errorEvent
		expectResult *cloudevents.Result
	}{
		"Successful request": {
			client:       &mockedSplunkClient{},
			expectEvent:  &errorEvent{},
			expectResult: &cloudevents.ResultACK,
		},
		"Failed request": {
			client: &mockedSplunkClient{
				err: assert.AnError,
			},
			expectEvent:  &errorEvent{Code: "adapter-process", Description: "assert.AnError general error for testing", Details: "failed to send event to HEC. Status code: 400"},
			expectResult: &cloudevents.ResultACK,
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			replier, _ := targetce.New("test", logtesting.TestLogger(t),
				targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeSplunkResponse))

			a := adapter{
				logger:       logtesting.TestLogger(t),
				ceClient:     adaptertest.NewTestClient(),
				spClient:     tc.client,
				defaultIndex: tDefaultIndex,
				replier:      replier,
			}

			// invoke event callback
			e, r := a.receive(context.Background(), newEvent(t))
			eE := &errorEvent{}

			e.DataAs(eE)
			assert.Lenf(t, tc.client.inputRecorder, 1, "Client records a single request")
			assert.Equal(t, &r, tc.expectResult)
			assert.Equal(t, eE, tc.expectEvent)
		})
	}
}

func newEvent(t *testing.T) cloudevents.Event {
	t.Helper()

	ce := cloudevents.NewEvent()
	ce.SetID("1234567890")
	ce.SetSource("test.source")
	ce.SetType("test.type")
	if err := ce.SetData(cloudevents.TextPlain, "Lorem Ipsum"); err != nil {
		t.Fatalf("Failed to set event data: %s", err)
	}

	return ce
}

type errorEvent struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Details     string `json:"details"`
}
