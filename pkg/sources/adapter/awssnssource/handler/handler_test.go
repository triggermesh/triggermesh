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

package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	logtesting "knative.dev/pkg/logging/testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"

	snsclient "github.com/triggermesh/triggermesh/pkg/sources/client/sns"
)

const (
	tMsgID      = "00000000-0000-0000-0000-000000000000"
	tMsgSubject = "My test message"
	tEventSrc   = "test.source"

	tTopicARN = "arn:aws:sns:us-fake-0:123456789012:MyTopic"
)

func TestHandler(t *testing.T) {
	t.Run("unsupported method", func(t *testing.T) {
		h := newTestHandler(t)

		req, err := http.NewRequest(http.MethodGet, "", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	})

	t.Run("missing message type header", func(t *testing.T) {
		h := newTestHandler(t)

		msgBody := strings.NewReader(tNotifMsg)
		req := newPostRequest(t, msgBody)

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("valid subscription confirmation", func(t *testing.T) {
		snsClient := &mockedSNSClient{}

		h := newTestHandler(t)
		h.snsClient = snsClient

		msgBody := strings.NewReader(tSubsConfirmMsg)
		req := newPostRequest(t, msgBody)
		req.Header.Set(headerMsgTypeKey, headerMsgTypeSubsConfirm)

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.True(t, snsClient.calledConfirmSubscribe)
	})

	t.Run("subscription confirmation fails", func(t *testing.T) {
		snsClient := &mockedSNSClient{
			failConfirmSubscription: true,
		}

		h := newTestHandler(t)
		h.snsClient = snsClient

		msgBody := strings.NewReader(tSubsConfirmMsg)
		req := newPostRequest(t, msgBody)
		req.Header.Set(headerMsgTypeKey, headerMsgTypeSubsConfirm)

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.True(t, snsClient.calledConfirmSubscribe)
		assert.Contains(t, rr.Body.String(), "Unable to confirm SNS subscription:")
	})

	t.Run("valid notification", func(t *testing.T) {
		ceClient := adaptertest.NewTestClient()

		h := newTestHandler(t)
		h.ceClient = ceClient

		msgBody := strings.NewReader(tNotifMsg)

		req := newPostRequest(t, msgBody)
		req.Header.Set(headerMsgTypeKey, headerMsgTypeNotification)
		req.Header.Set(headerMsgIDKey, tMsgID)

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		sentEvents := ceClient.Sent()
		require.Len(t, sentEvents, 1)

		event := sentEvents[0]
		assert.Equal(t, "com.amazon.sns.notification", event.Type())
		assert.Equal(t, tEventSrc, event.Source())
		assert.Equal(t, tMsgID, event.ID())
		assert.Equal(t, tMsgSubject, event.Subject())
	})

	t.Run("invalid notification payload", func(t *testing.T) {
		ceClient := adaptertest.NewTestClient()

		h := newTestHandler(t)
		h.ceClient = ceClient

		msgBody := strings.NewReader("{ not a JSON }")

		req := newPostRequest(t, msgBody)
		req.Header.Set(headerMsgTypeKey, headerMsgTypeNotification)
		req.Header.Set(headerMsgIDKey, tMsgID)

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Failed to parse notification:")

		require.Empty(t, ceClient.Sent())
	})
}

func newTestHandler(t *testing.T) *Handler {
	t.Helper()

	return &Handler{
		logger:   logtesting.TestLogger(t),
		eventSrc: tEventSrc,
	}
}

func newPostRequest(t *testing.T, body io.Reader) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, "", body)
	require.NoError(t, err)

	return req
}

type mockedSNSClient struct {
	snsclient.Client

	failConfirmSubscription bool
	calledConfirmSubscribe  bool
}

func (c *mockedSNSClient) ConfirmSubscription(*sns.ConfirmSubscriptionInput) (*sns.ConfirmSubscriptionOutput, error) {
	c.calledConfirmSubscribe = true

	if c.failConfirmSubscription {
		return nil, assert.AnError
	}

	return &sns.ConfirmSubscriptionOutput{
		SubscriptionArn: aws.String(tTopicARN + "/0123456789"),
	}, nil
}

const tNotifMsg = `{
  "Type": "` + headerMsgTypeNotification + `",
  "MessageId": "` + tMsgID + `",
  "TopicArn": "` + tTopicARN + `",
  "Subject": "` + tMsgSubject + `",
  "Message": "Hello world!",
  "Timestamp": "2012-05-02T00:54:06.655Z",
  "SignatureVersion": "1",
  "Signature": "EXAMPLEw6JRN...",
  "SigningCertURL": "https://sns.us-fake-0.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem",
  "UnsubscribeURL": "https://sns.us-fake-0.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-fake-0:123456789012:MyTopic:c9135db0-26c4-47ec-8998-413945fb5a96"
}`

const tSubsConfirmMsg = `{
  "Type": "` + headerMsgTypeSubsConfirm + `",
  "MessageId": "` + tMsgID + `",
  "Token": "2336412f37...",
  "TopicArn": "` + tTopicARN + `",
  "Message": "You have chosen to subscribe to the topic arn:aws:sns:us-fake-0:123456789012:MyTopic.\nTo confirm the subscription, visit the SubscribeURL included in this message.",
  "SubscribeURL": "https://sns.us-fake-0.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn:aws:sns:us-fake-0:123456789012:MyTopic&Token=2336412f37...",
  "Timestamp": "2012-04-26T20:45:04.751Z",
  "SignatureVersion": "1",
  "Signature": "EXAMPLEpH+DcEwjAPg8O9mY8dReBSwksfg2S7WKQcikcNKWLQjwu6A4VbeS0QHVCkhRS7fUQvi2egU3N858fiTDN6bkkOxYDVrY0Ad8L10Hs3zH81mtnPk5uvvolIC1CXGu43obcgFxeL3khZl8IKvO61GWB6jI9b5+gLPoBc1Q=",
  "SigningCertURL": "https://sns.us-fake-0.amazonaws.com/SimpleNotificationService-f3ecfb7224c7233fe7bb5f59f96de52f.pem"
}`
