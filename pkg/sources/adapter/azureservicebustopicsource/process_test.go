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

package azureservicebustopicsource

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	servicebus "github.com/Azure/azure-service-bus-go"
)

func TestProcessMessage(t *testing.T) {
	const ceType = "com.microsoft.azure.servicebus.message"
	const ceSource = "/resource/id/of/a/servicebus/topic"
	const ceID = "someMessageID"
	var ceTime = time.Unix(0, 0)

	testData := &servicebus.Message{
		ID: ceID,
		SystemProperties: &servicebus.SystemProperties{
			EnqueuedTime: &ceTime,
		},

		Data: sampleEvent,
	}

	msgPrcsr := &defaultMessageProcessor{
		ceSource: ceSource,
	}
	events, err := msgPrcsr.Process(testData)

	require.NoError(t, err)
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, ceID, event.ID())
	assert.Equal(t, ceType, event.Type())
	assert.Equal(t, ceSource, event.Source())
	assert.Equal(t, ceTime, event.Time())
}

// Generated using https://www.json-generator.com
var sampleEvent = []byte(`{
  "_id": "5fad5882028c6aafa3447b6e",
  "index": 0,
  "guid": "6d679c93-da46-40f5-914c-53fc263b7f98",
  "isActive": true,
  "balance": "$2,906.29",
  "picture": "http://placehold.it/32x32",
  "age": 26,
  "eyeColor": "brown",
  "name": {
    "first": "Jo",
    "last": "Murray"
  },
  "company": "ANIVET",
  "email": "jo.murray@anivet.name",
  "phone": "+1 (856) 573-3357",
  "address": "264 Vernon Avenue, Layhill, Arizona, 5744",
  "about": "Incididunt non sint nostrud veniam aliqua laborum veniam est in ut incididunt.",
  "registered": "Friday, February 28, 2020 12:40 AM",
  "latitude": "-79.885342",
  "longitude": "41.88282",
  "tags": [
    "quis",
    "sunt"
  ],
  "friends": [
    {
      "id": 0,
      "name": "Annabelle Hill"
    },
    {
      "id": 1,
      "name": "Isabel Delaney"
    },
    {
      "id": 2,
      "name": "Jessie Morris"
    }
  ],
  "greeting": "Hello, Jo! You have 8 unread messages.",
  "favoriteFruit": "banana"
}`)
