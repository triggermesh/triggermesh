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

// Package handler contains the logic for handling SNS notifications over HTTP.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	"knative.dev/pkg/logging/logkey"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	snsclient "github.com/triggermesh/triggermesh/pkg/sources/client/sns"
)

const (
	headerMsgTypeKey  = "X-Amz-Sns-Message-Type"
	headerMsgIDKey    = "X-Amz-Sns-Message-Id"
	headerTopicARNKey = "X-Amz-Sns-Topic-Arn"

	headerMsgTypeNotification = "Notification"
	headerMsgTypeSubsConfirm  = "SubscriptionConfirmation"
)

// Handler handles the HTTP notifications sent to a SNS topic.
type Handler struct {
	logger *zap.SugaredLogger

	ceClient cloudevents.Client
	eventSrc string
	sink     string

	snsClient snsclient.Client
}

// Check that Handler implements http.Handler.
var _ http.Handler = (*Handler)(nil)

// New returns an initialized Handler.
func New(src *v1alpha1.AWSSNSSource, logger *zap.SugaredLogger,
	ceClient cloudevents.Client, snsClient snsclient.Client) *Handler {

	return &Handler{
		logger: logger.With(zap.String(logkey.Key, src.Namespace+"/"+src.Name)),

		ceClient: ceClient,
		eventSrc: src.AsEventSource(),
		sink:     src.Status.SinkURI.String(),

		snsClient: snsClient,
	}
}

var eventType = v1alpha1.AWSEventType(sns.ServiceName, v1alpha1.AWSSNSGenericEventType)

// ServeHTTP implements http.Handler.
// Confirms subscriptions when the incoming payload is a subscription
// confirmation message. Otherwise, converts SNS notifications to CloudEvents
// and sends them to the configured event sink.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Unsupported method "+r.Method, http.StatusMethodNotAllowed)
		return
	}

	// NOTE(antoineco): one would typically check the value of the
	// Content-Type HTTP header here, but SNS sets it to "text/plain"
	// instead of "application/json", so we simply omit this validation.
	// https://docs.aws.amazon.com/sns/latest/dg/sns-message-and-json-formats.html

	switch msgType := r.Header.Get(headerMsgTypeKey); msgType {
	case headerMsgTypeNotification:
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			handleError("Failed to read request body", err, http.StatusInternalServerError, h.logger, w)
			return
		}

		h.logger.Debug("Request body: ", string(body))

		notif := &notification{}
		if err := json.Unmarshal(body, notif); err != nil {
			handleError("Failed to parse notification", err, http.StatusBadRequest, h.logger, w)
			return
		}

		event := cloudevents.NewEvent()
		event.SetType(eventType)
		event.SetSource(h.eventSrc)
		event.SetID(r.Header.Get(headerMsgIDKey))
		event.SetSubject(notif.Subject)

		if err := event.SetData(cloudevents.ApplicationJSON, json.RawMessage(body)); err != nil {
			handleError("Failed to set event data", err, http.StatusInternalServerError, h.logger, w)
			return
		}

		ctx := cloudevents.ContextWithTarget(context.Background(), h.sink)

		if result := h.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
			handleError("Failed to send CloudEvent", err, http.StatusInternalServerError, h.logger, w)
			return
		}

		h.logger.Debug("Successfully sent SNS notification: ", event)

	case headerMsgTypeSubsConfirm:
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			handleError("Failed to read request body", err, http.StatusInternalServerError, h.logger, w)
			return
		}

		h.logger.Debug("Request body: ", string(body))

		subsConfirm := &subscriptionConfirmation{}
		if err := json.Unmarshal(body, subsConfirm); err != nil {
			handleError("Failed to parse subscription confirmation", err, http.StatusBadRequest, h.logger, w)
			return
		}

		resp, err := h.snsClient.ConfirmSubscription(&sns.ConfirmSubscriptionInput{
			TopicArn: aws.String(r.Header.Get(headerTopicARNKey)),
			Token:    aws.String(subsConfirm.Token),
		})
		if err != nil {
			handleError("Unable to confirm SNS subscription", err, http.StatusInternalServerError, h.logger, w)
			return
		}

		h.logger.Debug("Successfully confirmed SNS subscription: ", *resp)

	default:
		http.Error(w, "Unrecognized message type "+msgType, http.StatusBadRequest)
	}
}

// handleError logs the given error and writes it to the given ResponseWriter.
func handleError(msg string, err error, httpCode int, logger *zap.SugaredLogger, w http.ResponseWriter) {
	logger.Errorw(msg, zap.Error(err))
	http.Error(w, fmt.Sprint(msg, ": ", err), httpCode)
}

// notification represents a SNS message of type Notification.
// https://docs.aws.amazon.com/sns/latest/dg/sns-message-and-json-formats.html#http-notification-json
type notification struct {
	Subject string
}

// subscriptionConfirmation represents a SNS message of type SubscriptionConfirmation
// https://docs.aws.amazon.com/sns/latest/dg/sns-message-and-json-formats.html#http-subscription-confirmation-json
type subscriptionConfirmation struct {
	Token string
}
