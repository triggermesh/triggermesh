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

package splunktarget

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	cloudevents "github.com/cloudevents/sdk-go/v2"

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
		expectResult cloudevents.Result
	}{
		"Successful request": {
			client:       &mockedSplunkClient{},
			expectResult: cloudevents.ResultACK,
		},
		"Failed request": {
			client: &mockedSplunkClient{
				err: assert.AnError,
			},
			expectResult: cloudevents.NewHTTPResult(http.StatusBadRequest,
				"failed to send event to HEC: %s", assert.AnError),
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			a := adapter{
				logger:       logtesting.TestLogger(t),
				ceClient:     adaptertest.NewTestClient(),
				spClient:     tc.client,
				defaultIndex: tDefaultIndex,
			}

			// invoke event callback
			res := a.receive(context.Background(), newEvent(t))

			assert.Lenf(t, tc.client.inputRecorder, 1, "Client records a single request")
			assert.EqualError(t, res, tc.expectResult.Error())
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

func TestCustomURLPath(t *testing.T) {
	testCases := map[string]struct {
		url      string
		expected string
	}{
		"Default URL": {
			url:      "https://mysplunk.example.com:8088",
			expected: "https://mysplunk.example.com:8088/services/collector/event/1.0",
		},
		"Default URL with trailing /": {
			url:      "https://mysplunk.example.com:8088/",
			expected: "https://mysplunk.example.com:8088/services/collector/event/1.0",
		},
		"Custom URL": {
			url:      "https://mysplunk.example.com:8088/services/collector/event",
			expected: "https://mysplunk.example.com:8088/services/collector/event",
		},
	}

	for name, tc := range testCases {
		//nolint:scopelint
		t.Run(name, func(t *testing.T) {
			u, err := url.Parse(tc.url)
			assert.NoError(t, err, "Parsing test URL")

			sc := newClient(*u, "", "", "", false)
			assert.Equal(t, tc.expected, sc.URL, "Unexpected URL at Splunk client")
		})
	}
}
