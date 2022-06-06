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

package cloudevents

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol"
	"github.com/google/uuid"

	"go.uber.org/zap"
)

// Error code common categories.
const (
	ErrorCodeEventContext      = "event-context"
	ErrorCodeRequestParsing    = "request-parsing"
	ErrorCodeRequestValidation = "request-validation"
	ErrorCodeAdapterProcess    = "adapter-process"
	ErrorCodeParseResponse     = "response-parsing"

	ErrorCodeCloudEventProcessing = "cloudevents-processing"
)

// CloudEvents extension names and values.
const (
	ExtensionCategory             = "category"
	ExtensionCategoryValueError   = "error"
	ExtensionCategoryValueSuccess = "success"
)

// PayloadPolicy defines when to
// return payloads with the response.
type PayloadPolicy string

// Payload policy choices
const (
	PayloadPolicyAlways PayloadPolicy = "always"
	PayloadPolicyErrors PayloadPolicy = "errors"
	PayloadPolicyNever  PayloadPolicy = "never"
)

// Replier helps normalizing CloudEvent responses.
type Replier struct {
	// Functions that create standard header values.
	responseType        ResponseHeaderValue
	responseSource      ResponseHeaderValue
	responseContentType ResponseHeaderValue

	// Functions that create standard header
	// values for errors.
	responseErrorType        ResponseHeaderValue
	responseErrorContentType ResponseHeaderValue

	// Optional functions that modify output events.
	responseOptions []EventResponseOption

	// When to return payloads with the response.
	payloadPolicy PayloadPolicy

	logger *zap.SugaredLogger
}

// New returns a replier instance.
// Optional ReplierOption set can be returned to customize the replier behavior.
func New(targetName string, logger *zap.SugaredLogger, opts ...ReplierOption) (*Replier, error) {
	cer := &Replier{

		// Fill default values for standard headers.
		responseType:        SuffixResponseType(".response"),
		responseContentType: StaticResponse(cloudevents.ApplicationJSON),
		responseSource:      StaticResponse(targetName),

		// Error response type and content-type will use the same functions
		// as the success response if not explicitly set.

		payloadPolicy: PayloadPolicyAlways,

		logger: logger,
	}

	for _, opt := range opts {
		if err := opt(cer); err != nil {
			return nil, err
		}
	}

	return cer, nil
}

// Ack when processing went ok and there is no payload as a response.
func (r *Replier) Ack() (*cloudevents.Event, cloudevents.Result) {
	return nil, cloudevents.ResultACK
}

// ErrorKnativeManaged lets the knative channel manage errors. This is intended when
// we rely on knative platform retries and dead letter queues.
func (r *Replier) ErrorKnativeManaged(event *cloudevents.Event, err error) (*cloudevents.Event, cloudevents.Result) {
	summary := summarizeEvent(event)
	r.logger.Errorw("Retriable (kn) error", zap.Error(err), zap.String("event", summary))
	return nil, cloudevents.NewResult("retriable (kn) error at event %s: %w", summary, err)
}

// Ok when returning a payload upon success.
func (r *Replier) Ok(in *cloudevents.Event, payload interface{}, opts ...EventResponseOption) (*cloudevents.Event, cloudevents.Result) {
	// Skip responses if not configured to send
	// them on success.
	if r.payloadPolicy != PayloadPolicyAlways {
		return r.Ack()
	}

	out := cloudevents.NewEvent(cloudevents.VersionV1)

	rt, err := r.responseType(in)
	if err != nil {
		return r.Error(in, ErrorCodeCloudEventProcessing, fmt.Errorf("error choosing response type: %w", err), nil)
	}
	if err = out.Context.SetType(rt); err != nil {
		return r.Error(in, ErrorCodeCloudEventProcessing, fmt.Errorf("error setting response type: %w", err), nil)
	}

	rs, err := r.responseSource(in)
	if err != nil {
		return r.Error(in, ErrorCodeCloudEventProcessing, fmt.Errorf("error choosing response source: %w", err), nil)
	}
	if err = out.Context.SetSource(rs); err != nil {
		return r.Error(in, ErrorCodeCloudEventProcessing, fmt.Errorf("error setting response source: %w", err), nil)
	}

	err = out.Context.SetExtension(ExtensionCategory, ExtensionCategoryValueSuccess)
	if err != nil {
		return r.Error(in, ErrorCodeCloudEventProcessing,
			fmt.Errorf("error setting category header at response event: %w", err), map[string]interface{}{"payload": payload})
	}

	opts = append(r.responseOptions, opts...)

	for _, opt := range opts {
		if err = opt(in, &out); err != nil {
			r.logger.Errorw("Error applying response options at reply", zap.Error(err))
		}
	}

	// Set the ID if no option have already set it.
	if len(out.ID()) == 0 {
		out.SetID(uuid.New().String())
	}

	rct, err := r.responseContentType(in)
	if err != nil {
		return r.Error(in, ErrorCodeCloudEventProcessing, fmt.Errorf("error choosing response content-type: %w", err), nil)
	}

	err = out.SetData(rct, payload)
	if err != nil {
		return r.Error(in, ErrorCodeCloudEventProcessing,
			fmt.Errorf("error setting response payload at response event: %w", err), map[string]interface{}{"payload": payload})
	}

	return &out, cloudevents.ResultACK
}

// EventError is returned as the payload when an error occurs.
type EventError struct {
	// Code is an identifiable moniker that classifies the error.
	Code        string
	Description string
	// Details that contain arbitrary data about the error.
	Details interface{}
}

// Error replies with an error payload but dismisses the knative channel error management
// (retries and dead letter queues). This should be used when retrying would result in the
// same outcome, and when we want users to explicitly manage this error instead of relying on
// dead letter queue channel.
func (r *Replier) Error(in *cloudevents.Event, code string, reportedErr error, details interface{}, opts ...EventResponseOption) (*cloudevents.Event, cloudevents.Result) {
	r.logger.Errorw("Processing error",
		zap.Error(reportedErr),
		zap.Any("in-event", *in),
		zap.Any("details", details),
	)
	if r.payloadPolicy == PayloadPolicyNever {
		return nil, protocol.ResultACK
	}

	out := cloudevents.NewEvent(cloudevents.VersionV1)

	// If response error type function is not set, use the
	// same function as the success case.
	errorTypeFn := r.responseErrorType
	if errorTypeFn == nil {
		errorTypeFn = r.responseType
	}

	rt, err := errorTypeFn(in)
	if err != nil {
		r.logger.Errorw("Error choosing error response type", zap.Error(err))
		return nil, cloudevents.ResultACK
	}
	err = out.Context.SetType(rt)
	if err != nil {
		r.logger.Errorw("Could not set event type at error response", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	rs, err := r.responseSource(in)
	if err != nil {
		r.logger.Errorw("Error choosing error response source", zap.Error(err))
		return nil, cloudevents.ResultACK
	}
	err = out.Context.SetSource(rs)
	if err != nil {
		r.logger.Errorw("Could not set event source at error response", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	err = out.Context.SetExtension(ExtensionCategory, ExtensionCategoryValueError)
	if err != nil {
		r.logger.Errorw("Could not set event category extension at error response", zap.Error(err))
		return nil, cloudevents.ResultACK
	}

	opts = append(r.responseOptions, opts...)

	for _, opt := range opts {
		if err = opt(in, &out); err != nil {
			r.logger.Errorw("Error applying response options at error reply", zap.Error(err))
		}
	}

	// Set the ID if no option have already set it.
	if len(out.ID()) == 0 {
		out.SetID(uuid.New().String())
	}

	// If response error content-type function is not set, use the
	// same function as the success case.
	errorContentTypeFn := r.responseErrorContentType
	if errorContentTypeFn == nil {
		errorContentTypeFn = r.responseContentType
	}

	rct, err := errorContentTypeFn(in)
	if err != nil {
		return r.Error(in, ErrorCodeCloudEventProcessing, fmt.Errorf("error choosing error response content-type: %w", err), nil)
	}

	evErr := &EventError{
		Code:    code,
		Details: details,
	}

	// Reported error is expected to be informed, but in case it is not
	// we need to protect against panic.
	if reportedErr != nil {
		evErr.Description = reportedErr.Error()
	}

	if err = out.SetData(rct, evErr); err != nil {
		r.logger.Errorw("Could not set error payload at response CloudEvent", zap.Error(err))
	}

	return &out, cloudevents.ResultACK
}

// cloudEventSummary is used to generate formatted serializations
// of a CloudEvent.
type cloudEventSummary struct {
	ID      string `json:"id"`
	CEType  string `json:"type"`
	Source  string `json:"source"`
	Subject string `json:"subject,omitempty"`
}

// summarizeEvent creates a string that contains representative
// header information for an event.
func summarizeEvent(event *cloudevents.Event) string {
	ces := &cloudEventSummary{
		ID:      event.ID(),
		CEType:  event.Context.GetType(),
		Source:  event.Context.GetSource(),
		Subject: event.Context.GetSubject(),
	}

	b, err := json.Marshal(ces)
	if err != nil {
		// There should be no errors with the serialization,
		// adding this for very improbable cases.
		return fmt.Sprintf(`{"id":%q, "type":%q, "source":%q, "subject":%q`,
			event.ID(), event.Context.GetType(), event.Context.GetSource(), event.Context.GetSubject())
	}

	return string(b)
}
