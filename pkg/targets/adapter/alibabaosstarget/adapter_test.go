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

package alibabaosstarget

import (
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"
)

const (
	tCloudEventID     = "ce-abcd-0123"
	tCloudEventType   = "ce.test.type"
	tCloudEventSource = "ce.test.source"

	tXML1        = `<note><to>Tove</to></note>`
	tJSONOutput1 = "{\"note\": {\"to\": \"Tove\"}}\n"

	tFalseXML         = `"this is not xml"`
	tFalseXMLResponse = `{"Code":"request-validation","Description":"invalid XML","Details":null}`
)

func TestAlibaba(t *testing.T) {
	testCases := map[string]struct {
		inEvent     cloudevents.Event
		expectEvent cloudevents.Event
	}{
		"Puts Object": {
			inEvent:     newCloudEvent(tXML1, cloudevents.ApplicationXML),
			expectEvent: newCloudEvent(tJSONOutput1, cloudevents.ApplicationJSON),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ceClient := adaptertest.NewTestClient()
			a := &ossAdapter{
				logger:   loggingtesting.TestLogger(t),
				ceClient: ceClient,
				bucket:  "test-bucket",
				oClient: nil,
				pof: ,
			}


		})
	}
}

func putObjectMock(oclient *oss.Client, objectKey string, reader io.Reader, event cloudevents.Event, bucketName string) error {
	// check that a bucket was provided

	// check 
	
	// bucket, err := oclient.Bucket(bucketName)
	// if err != nil {
	// 	return err
	// }

	// if bucket == nil {
	// 	return err
	// }

	// if err = bucket.PutObject(event.ID(), bytes.NewReader(event.Data())); err != nil {
	// 	return err
	// }

	return nil
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

	return event
}
