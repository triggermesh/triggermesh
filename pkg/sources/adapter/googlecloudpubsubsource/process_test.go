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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cloud.google.com/go/pubsub"
)

func TestProcessMessageDefault(t *testing.T) {
	const ceSource = "fake.source"
	const ceType = "fake.type"

	testData := fakePubSubMessage()

	msgPrcsr := &defaultMessageProcessor{
		ceSource: ceSource,
		ceType:   ceType,
	}

	events, err := msgPrcsr.Process(testData)

	require.NoError(t, err)
	require.Len(t, events, 1)

	event := events[0]

	assert.Equal(t, "000", event.ID())
	assert.Equal(t, time.Unix(0, 0), event.Time())
	assert.Equal(t, ceSource, event.Source())
	assert.Equal(t, ceType, event.Type())

	eventExts := event.Extensions()

	assert.Contains(t, eventExts, "pubsubmsgsomething")
	assert.Contains(t, eventExts, "pubsubmsgmixedcase")
	assert.Contains(t, eventExts, "pubsubmsginvalidchars")
	assert.Contains(t, eventExts, "pubsubmsgsubject")

	assert.NotContains(t, eventExts, "something")
	assert.NotContains(t, eventExts, "mixedCase")
	assert.NotContains(t, eventExts, "invalid-chars")
	assert.NotContains(t, eventExts, "subject")
}

// fakePubSubMessage returns a Pub/Sub message to be used in tests.
func fakePubSubMessage() *pubsub.Message {
	return &pubsub.Message{
		ID:          "000",
		PublishTime: time.Unix(0, 0),
		Attributes: map[string]string{
			"something":     "value",
			"mixedCase":     "value",
			"invalid-chars": "value",
			"subject":       "value", // reserved CE attribute
		},
		Data: []byte(`{"msg": "Hello, World!"}`),
	}
}
