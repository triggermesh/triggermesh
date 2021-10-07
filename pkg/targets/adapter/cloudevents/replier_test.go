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

package cloudevents

import (
	"errors"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	zapt "go.uber.org/zap/zaptest"
)

const (
	tBridge     = "test-bridge"
	tTargetName = "test-target"

	tEventID      = "in-1234"
	tEventType    = "in.type"
	tBadEventType = "bad.type"
	tEventSource  = "in.source"
	tEventSubject = "in subject"
	tEventData    = `{"hello":"world"}`

	tGenerated = "generated"

	tOutID            = "out-1234"
	tOutAlternateType = "out.type"
	tOutSubject       = "test subject"

	tErrorCode = ErrorCodeCloudEventProcessing
)

var (
	tPayload     = []byte(`bye-world`)
	tOutType     = tEventType + ".response"
	tOutBadType  = tBadEventType + ".response"
	tOutSource   = tTargetName
	tMappedTypes = map[string]string{tEventType: tOutAlternateType}

	errTest = errors.New("reported error")
)

func TestAckReplies(t *testing.T) {
	logger := zapt.NewLogger(t).Sugar()
	replier, err := New(tTargetName, logger)
	if err != nil {
		assert.Error(t, err, "Unexpected error building replier")
	}

	ev, res := replier.Ack()
	assert.Nil(t, ev, "Unexpected non nil reply from Ack")
	assert.Equal(t, cloudevents.ResultACK, res, "Unexpected non nil reply from Ack")
}

func TestOkReplies(t *testing.T) {
	logger := zapt.NewLogger(t).Sugar()
	tc := map[string]struct {
		// Replier options
		replierOptions []ReplierOption

		// Incoming event
		in *cloudevents.Event

		// Target provided data
		payload interface{}

		// Response options

		eventResponseOptions []EventResponseOption

		// Expected data
		expectedNilEvent   bool
		expectedID         string
		expectedType       string
		expectedSource     string
		expectedSubject    string
		expectedExtensions map[string]interface{}
	}{
		"default replier": {
			in: createFakeEvent(tEventType),

			payload: tPayload,

			expectedNilEvent:   false,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedExtensions: createSuccessCategoryExtension(),
		},
		"payload on errors policy replier": {
			replierOptions: []ReplierOption{ReplierWithPayloadPolicy(PayloadPolicyErrors)},
			in:             createFakeEvent(tEventType),
			payload:        tPayload,

			expectedNilEvent:   true,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedExtensions: createSuccessCategoryExtension(),
		},
		"payload never policy replier": {
			replierOptions: []ReplierOption{ReplierWithPayloadPolicy(PayloadPolicyNever)},
			in:             createFakeEvent(tEventType),
			payload:        tPayload,

			expectedNilEvent:   true,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedExtensions: createSuccessCategoryExtension(),
		},
		"custom ID replier": {
			in:                   createFakeEvent(tEventType),
			payload:              tPayload,
			eventResponseOptions: []EventResponseOption{ResponseWithID(tOutID)},

			expectedNilEvent:   true,
			expectedID:         tOutID,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedExtensions: createSuccessCategoryExtension(),
		},
		"type from static replier": {
			in:             createFakeEvent(tEventType),
			payload:        tPayload,
			replierOptions: []ReplierOption{ReplierWithStaticResponseType(tOutAlternateType)},

			expectedNilEvent:   true,
			expectedType:       tOutAlternateType,
			expectedSource:     tOutSource,
			expectedExtensions: createSuccessCategoryExtension(),
		},
		"type from maped replier": {
			in:             createFakeEvent(tEventType),
			payload:        tPayload,
			replierOptions: []ReplierOption{ReplierWithMappedResponseType(tMappedTypes)},

			expectedNilEvent:   true,
			expectedType:       tOutAlternateType,
			expectedSource:     tOutSource,
			expectedExtensions: createSuccessCategoryExtension(),
		},
		"stateful headers replier": {
			in:             createFakeEvent(tEventType),
			payload:        tPayload,
			replierOptions: []ReplierOption{ReplierWithStatefulHeaders(tBridge)},

			expectedNilEvent: false,
			expectedType:     tOutType,
			expectedSource:   tOutSource,
			expectedExtensions: map[string]interface{}{
				ExtensionCategory:              ExtensionCategoryValueSuccess,
				StatefulWorkflowHeader:         tBridge,
				StatefulWorkflowInstanceHeader: tGenerated,
			},
		},
		"processed headers replier": {
			in:             createFakeEvent(tEventType),
			payload:        tPayload,
			replierOptions: []ReplierOption{ReplierWithProcessedHeaders()},

			expectedNilEvent: false,
			expectedType:     tOutType,
			expectedSource:   tOutSource,
			expectedExtensions: map[string]interface{}{
				ExtensionCategory:     ExtensionCategoryValueSuccess,
				ProcessedTypeHeader:   tEventType,
				ProcessedSourceHeader: tEventSource,
				ProcessedIDHeader:     tEventID,
			},
		},
		"processed and stateful headers replier": {
			in:      createFakeEvent(tEventType),
			payload: tPayload,
			replierOptions: []ReplierOption{
				ReplierWithProcessedHeaders(),
				ReplierWithStatefulHeaders(tBridge)},

			expectedNilEvent: false,
			expectedType:     tOutType,
			expectedSource:   tOutSource,
			expectedExtensions: map[string]interface{}{
				ExtensionCategory:              ExtensionCategoryValueSuccess,
				StatefulWorkflowHeader:         tBridge,
				StatefulWorkflowInstanceHeader: tGenerated,
				ProcessedTypeHeader:            tEventType,
				ProcessedSourceHeader:          tEventSource,
				ProcessedIDHeader:              tEventID,
			},
		},
		"reply with subject headers replier": {
			in:                   createFakeEvent(tEventType),
			payload:              tPayload,
			eventResponseOptions: []EventResponseOption{ResponseWithSubject(tOutSubject)},

			expectedNilEvent:   false,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedSubject:    tOutSubject,
			expectedExtensions: createSuccessCategoryExtension(),
		},
		"mapped replier with no matching key": {
			in:             createFakeEvent(tBadEventType),
			payload:        tPayload,
			replierOptions: []ReplierOption{ReplierWithMappedResponseType(tMappedTypes)},

			expectedNilEvent:   true,
			expectedType:       tOutBadType,
			expectedSource:     tOutSource,
			expectedExtensions: createSuccessCategoryExtension(),
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {

			replier, err := New(tTargetName, logger, c.replierOptions...)
			if err != nil {
				assert.Error(t, err, "Unexpected error building replier")
			}

			out, res := replier.Ok(c.in, c.payload, c.eventResponseOptions...)

			assert.Equal(t, cloudevents.ResultACK, res, "Unexpected response result")
			if out == nil {
				assert.True(t, c.expectedNilEvent, "Got unexpected nil event response")
				return
			}

			assert.NotEmpty(t, out.Context.GetID(), "Response event ID should not be empty")
			if c.expectedID != "" {
				assert.Equal(t, c.expectedID, out.Context.GetID(), "Unexpected response ID")
			}
			assert.Equal(t, c.payload, out.Data(), "Unexpected response payload")
			assert.Equal(t, c.expectedType, out.Context.GetType(), "Unexpected response type")
			assert.Equal(t, c.expectedSource, out.Context.GetSource(), "Unexpected response source")
			assert.Equal(t, c.expectedSubject, out.Context.GetSubject(), "Unexpected response subecjt")

			exts := out.Context.GetExtensions()

			// generated headers need to be avoided, we are setting them
			//  to generic values for this comparison
			setGenericValuesAtGeneratedHeaders(exts)

			assert.Equal(t, c.expectedExtensions, exts)

		})
	}
}

func TestErrorReplies(t *testing.T) {
	logger := zapt.NewLogger(t).Sugar()
	tc := map[string]struct {
		// Replier options.
		replierOptions []ReplierOption

		// Incoming event.
		in *cloudevents.Event

		// Target provided data.
		code          string
		reportedError error
		details       interface{}

		// Response options.

		eventResponseOptions []EventResponseOption

		// Expected data.
		expectedNilEvent   bool
		expectedID         string
		expectedType       string
		expectedSource     string
		expectedSubject    string
		expectedExtensions map[string]interface{}
	}{
		"default replier": {
			in:            createFakeEvent(tEventType),
			code:          tErrorCode,
			reportedError: errTest,

			expectedNilEvent:   false,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedExtensions: createErrorCategoryExtension(),
		},
		"payload on errors policy replier": {
			replierOptions: []ReplierOption{ReplierWithPayloadPolicy(PayloadPolicyErrors)},
			in:             createFakeEvent(tEventType),
			code:           tErrorCode,
			reportedError:  errTest,

			expectedNilEvent:   false,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedExtensions: createErrorCategoryExtension(),
		},
		"payload never policy replier": {
			replierOptions: []ReplierOption{ReplierWithPayloadPolicy(PayloadPolicyNever)},
			in:             createFakeEvent(tEventType),
			code:           tErrorCode,
			reportedError:  errTest,

			expectedNilEvent:   true,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedExtensions: createErrorCategoryExtension(),
		},
		"error with fields": {
			in:            createFakeEvent(tEventType),
			code:          tErrorCode,
			reportedError: errTest,
			details:       createErrorFields(),

			expectedNilEvent:   false,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedExtensions: createErrorCategoryExtension(),
		},
		"custom ID error": {
			in:                   createFakeEvent(tEventType),
			code:                 tErrorCode,
			reportedError:        errTest,
			eventResponseOptions: []EventResponseOption{ResponseWithID(tOutID)},

			expectedNilEvent:   false,
			expectedID:         tOutID,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedExtensions: createErrorCategoryExtension(),
		},
		"type from static replier": {
			in:             createFakeEvent(tEventType),
			code:           tErrorCode,
			reportedError:  errTest,
			replierOptions: []ReplierOption{ReplierWithStaticErrorResponseType(tOutAlternateType)},

			expectedNilEvent:   false,
			expectedType:       tOutAlternateType,
			expectedSource:     tOutSource,
			expectedExtensions: createErrorCategoryExtension(),
		},
		"type from static replier, not equal to success type": {
			in:            createFakeEvent(tEventType),
			code:          tErrorCode,
			reportedError: errTest,
			replierOptions: []ReplierOption{
				ReplierWithStaticErrorResponseType(tOutAlternateType),
				ReplierWithStaticResponseType("success.type")},

			expectedNilEvent:   false,
			expectedType:       tOutAlternateType,
			expectedSource:     tOutSource,
			expectedExtensions: createErrorCategoryExtension(),
		},
		"type from maped replier": {
			in:             createFakeEvent(tEventType),
			code:           tErrorCode,
			reportedError:  errTest,
			replierOptions: []ReplierOption{ReplierWithMappedErrorResponseType(tMappedTypes)},

			expectedNilEvent:   false,
			expectedType:       tOutAlternateType,
			expectedSource:     tOutSource,
			expectedExtensions: createErrorCategoryExtension(),
		},
		"stateful headers replier": {
			in:             createFakeEvent(tEventType),
			code:           tErrorCode,
			reportedError:  errTest,
			replierOptions: []ReplierOption{ReplierWithStatefulHeaders(tBridge)},

			expectedNilEvent: false,
			expectedType:     tOutType,
			expectedSource:   tOutSource,
			expectedExtensions: map[string]interface{}{
				ExtensionCategory:              ExtensionCategoryValueError,
				StatefulWorkflowHeader:         tBridge,
				StatefulWorkflowInstanceHeader: tGenerated,
			},
		},
		"processed headers replier": {
			in:             createFakeEvent(tEventType),
			code:           tErrorCode,
			reportedError:  errTest,
			replierOptions: []ReplierOption{ReplierWithProcessedHeaders()},

			expectedNilEvent: false,
			expectedType:     tOutType,
			expectedSource:   tOutSource,
			expectedExtensions: map[string]interface{}{
				ExtensionCategory:     ExtensionCategoryValueError,
				ProcessedTypeHeader:   tEventType,
				ProcessedSourceHeader: tEventSource,
				ProcessedIDHeader:     tEventID,
			},
		},
		"processed and stateful headers replier": {
			in:            createFakeEvent(tEventType),
			code:          tErrorCode,
			reportedError: errTest,
			replierOptions: []ReplierOption{
				ReplierWithProcessedHeaders(),
				ReplierWithStatefulHeaders(tBridge)},

			expectedNilEvent: false,
			expectedType:     tOutType,
			expectedSource:   tOutSource,
			expectedExtensions: map[string]interface{}{
				ExtensionCategory:              ExtensionCategoryValueError,
				StatefulWorkflowHeader:         tBridge,
				StatefulWorkflowInstanceHeader: tGenerated,
				ProcessedTypeHeader:            tEventType,
				ProcessedSourceHeader:          tEventSource,
				ProcessedIDHeader:              tEventID,
			},
		},
		"error reply with subject headers replier ": {
			in:                   createFakeEvent(tEventType),
			code:                 tErrorCode,
			reportedError:        errTest,
			eventResponseOptions: []EventResponseOption{ResponseWithSubject(tOutSubject)},

			expectedNilEvent:   false,
			expectedType:       tOutType,
			expectedSource:     tOutSource,
			expectedSubject:    tOutSubject,
			expectedExtensions: createErrorCategoryExtension(),
		},
		"error reply no matching event types ": {
			in:                   createFakeEvent(tBadEventType),
			code:                 tErrorCode,
			reportedError:        errTest,
			eventResponseOptions: []EventResponseOption{ResponseWithSubject(tOutSubject)},

			expectedNilEvent:   false,
			expectedType:       tOutBadType,
			expectedSource:     tOutSource,
			expectedSubject:    tOutSubject,
			expectedExtensions: createErrorCategoryExtension(),
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {

			replier, err := New(tTargetName, logger, c.replierOptions...)
			if err != nil {
				assert.Error(t, err, "Unexpected error building replier")
			}

			out, res := replier.Error(c.in, c.code, c.reportedError, c.details, c.eventResponseOptions...)

			assert.Equal(t, cloudevents.ResultACK, res, "Unexpected response result")
			if out == nil {
				assert.True(t, c.expectedNilEvent, "Got unexpected nil event response")
				return
			}

			assert.NotEmpty(t, out.Context.GetID(), "Response event ID should not be empty")
			if c.expectedID != "" {
				assert.Equal(t, c.expectedID, out.Context.GetID(), "Unexpected response ID")
			}

			outData := &EventError{}
			err = out.DataAs(outData)
			assert.Nil(t, err, "Returned data is not an EventError")

			expectedData := &EventError{
				Code:        c.code,
				Description: c.reportedError.Error(),
				Details:     c.details,
			}

			assert.Equal(t, expectedData, outData, "Unexpected response payload")
			assert.Equal(t, c.expectedType, out.Context.GetType(), "Unexpected response type")
			assert.Equal(t, c.expectedSource, out.Context.GetSource(), "Unexpected response source")
			assert.Equal(t, c.expectedSubject, out.Context.GetSubject(), "Unexpected response subecjt")

			exts := out.Context.GetExtensions()

			// Generated headers need to be avoided, we are setting them
			// to generic values for this comparison.
			setGenericValuesAtGeneratedHeaders(exts)

			assert.Equal(t, c.expectedExtensions, exts)

		})
	}
}

func createFakeEvent(eventType string) *cloudevents.Event {
	event := cloudevents.NewEvent(cloudevents.VersionV1)

	event.SetID(tEventID)
	event.SetType(eventType)
	event.SetSource(tEventSource)
	event.SetSubject(tEventSubject)
	if err := event.SetData(cloudevents.ApplicationJSON, tEventData); err != nil {
		panic(err)
	}

	return &event
}

func createSuccessCategoryExtension() map[string]interface{} {
	return map[string]interface{}{
		ExtensionCategory: ExtensionCategoryValueSuccess,
	}
}

func createErrorCategoryExtension() map[string]interface{} {
	return map[string]interface{}{
		ExtensionCategory: ExtensionCategoryValueError,
	}
}

func setGenericValuesAtGeneratedHeaders(h map[string]interface{}) {
	genericHeaders := []string{StatefulWorkflowInstanceHeader}
	for _, gh := range genericHeaders {
		if _, ok := h[gh]; ok {
			h[gh] = tGenerated
		}
	}
}

func createErrorFields() map[string]interface{} {
	return map[string]interface{}{
		"keyA": "ValueA",
		"keyB": "ValueB",
	}
}
