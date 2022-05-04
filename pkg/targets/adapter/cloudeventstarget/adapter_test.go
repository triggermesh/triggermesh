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

package cloudeventstarget

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"

	fakefs "github.com/triggermesh/triggermesh/pkg/adapter/fs/fake"
	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

func TestCloudEventsDispatch(t *testing.T) {
	logger := loggingtesting.TestLogger(t)

	eventRight := cloudevents.NewEvent(cloudevents.VersionV1)
	eventRight.SetType("type")
	eventRight.SetSource("source")
	eventRight.SetID("id")
	eventRight.SetTime(time.Now())

	// Using the event type `unit.sendFail` makes the event fail when being sent.
	// See: https://github.com/knative/eventing/blob/ec36c8637ddef2333fdc80bff8a963ffcfcc5059/pkg/adapter/v2/test/test_client.go#L67-L95
	eventWrong := cloudevents.NewEvent(cloudevents.VersionV1)
	eventWrong.SetType("unit.sendFail")
	eventWrong.SetSource("source")
	eventWrong.SetID("id")
	eventWrong.SetTime(time.Now())

	testCases := map[string]struct {
		senderReady bool
		ce          cloudevents.Event

		expectedError bool
	}{
		"Test succeed": {
			senderReady: true,
			ce:          eventRight,

			expectedError: false,
		},
		"Test failed": {
			senderReady: true,
			ce:          eventWrong,

			expectedError: true,
		},
		"Test CE client not configured": {
			senderReady: false,
			ce:          eventRight,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			mt := &pkgadapter.MetricTag{
				ResourceGroup: targets.CloudEventsTargetResource.String(),
				Namespace:     "ns-test",
				Name:          "name-test",
			}

			adapter := &ceAdapter{
				fileWatcher:  fakefs.NewFileWatcher(),
				listenClient: adaptertest.NewTestClient(),
				logger:       logger,
				m:            sync.RWMutex{},
				sr:           metrics.MustNewEventProcessingStatsReporter(mt),
			}

			var ceClientSender *adaptertest.TestCloudEventsClient = nil
			if tc.senderReady {
				ceClientSender = adaptertest.NewTestClient()
				adapter.senderClient = ceClientSender
			}

			var eventData adaptertest.EventData
			bytes, _ := json.Marshal(tc.ce)
			if err := json.Unmarshal(bytes, &eventData); err != nil {
				t.Fatal(err)
			}

			res := adapter.dispatch(ctx, tc.ce)

			switch {
			case tc.expectedError:
				assert.True(t, cloudevents.IsNACK(res), "dispatch result was %q", res.Error())

			case !tc.senderReady:
				assert.Error(t, res, "Adapter sender not ready should lead to an error")

			default:
				ceSent := ceClientSender.Sent()
				require.Equal(t, 1, len(ceSent), "Produced an unexpected number of CloudEvents")
				assert.Equal(t, tc.ce, ceSent[0], "CloudEvent received does not match produced")
			}
		})
	}
}
