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

package googlecloudpubsubsource

import (
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

func TestGCSProcessMessage(t *testing.T) {
	var testCases = []struct {
		pubSubEventType   string
		expectedEventType string
	}{
		{
			pubSubEventType:   "OBJECT_FINALIZE",
			expectedEventType: v1alpha1.GoogleCloudStorageFinalizeEventType,
		}, {
			pubSubEventType:   "OBJECT_METADATA_UPDATE",
			expectedEventType: v1alpha1.GoogleCloudStorageUpdateEventType,
		}, {
			pubSubEventType:   "OBJECT_DELETE",
			expectedEventType: v1alpha1.GoogleCloudStorageDeleteEventType,
		}, {
			pubSubEventType:   "OBJECT_ARCHIVE",
			expectedEventType: v1alpha1.GoogleCloudStorageArchiveEventType,
		}, {
			pubSubEventType:   "unhandled PubSub type",
			expectedEventType: v1alpha1.GoogleCloudStorageGenericEventType,
		}, {
			pubSubEventType:   "",
			expectedEventType: v1alpha1.GoogleCloudStorageGenericEventType,
		},
	}

	const ceSource = "fake.source"
	testPayload := fakePubSubGCSMessage()
	gcsPrcsr := &gcsMessageProcessor{
		ceSource: ceSource,
	}

	for _, testCase := range testCases {
		if testCase.pubSubEventType != "" {
			testPayload.Attributes["eventType"] = testCase.pubSubEventType
		}

		events, err := gcsPrcsr.Process(testPayload)

		require.NoError(t, err)
		require.Len(t, events, 1)

		event := events[0]
		assert.Equal(t, "000", event.ID())
		assert.Equal(t, time.Unix(0, 0), event.Time())
		assert.Equal(t, ceSource, event.Source())
		assert.Equal(t, testCase.expectedEventType, event.Type())
	}
}

// fakePubSubMessage returns a Pub/Sub message to be used in tests.
func fakePubSubGCSMessage() *pubsub.Message {
	return &pubsub.Message{
		ID:          "000",
		PublishTime: time.Unix(0, 0),
		Attributes:  make(map[string]string),
		Data:        []byte(`{"msg": "Hello, World!"}`),
	}
}
