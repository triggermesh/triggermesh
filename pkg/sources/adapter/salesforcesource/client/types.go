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

package client

import (
	"context"
	"encoding/json"
	"time"
)

// CommonResponse for all structures at Bayeux protocol.
type CommonResponse struct {
	Channel    string `json:"channel"`
	ClientID   string `json:"clientId"`
	Successful bool   `json:"successful"`
	Error      string `json:"error,omitempty"`
}

// HandshakeResponse for Bayeux protocol.
type HandshakeResponse struct {
	CommonResponse           `json:",inline"`
	Version                  string
	MinimumVersion           string
	SupportedConnectionTypes []string
	Advice                   Advice          `json:"advice,omitempty"`
	Extension                json.RawMessage `json:"ext,omitempty"`
}

// ConnectResponse for Bayeux protocol
type ConnectResponse struct {
	CommonResponse `json:",inline"`
	Data           struct {
		Event struct {
			CreatedDate time.Time `json:"createdDate,omitempty"`
			ReplayID    int64     `json:"replayId,omitempty"`
			Type        string    `json:"type,omitempty"`
		} `json:"event,omitempty"`
		Schema  string          `json:"schema,omitempty"`
		SObject json.RawMessage `json:"sobject,omitempty"`
		Payload json.RawMessage `json:"payload,omitempty"`
	} `json:"data,omitempty"`
	Advice Advice `json:"advice,omitempty"`
}

// SubscriptionResponse for Bayeux protocol
type SubscriptionResponse struct {
	CommonResponse `json:",inline"`
	Subscription   string `json:"subscription,omitempty"`
	Advice         Advice `json:"advice,omitempty"`
}

// Advice reusable structure for Bayeux protocol
type Advice struct {
	Reconnect string `json:"reconnect,omitempty"`
	Timeout   int    `json:"timeout,omitempty"`
	Interval  int    `json:"interval,omitempty"`
}

// ChangeDataCapturePayload is a partial structure of the CDC message
// that the adapter needs to figure out the subject for the CloudEvent.
type ChangeDataCapturePayload struct {
	ChangeEventHeader struct {
		EntityName string
		ChangeType string
	} `json:"ChangeEventHeader"`
}

// PushTopicSObject is a partial structure of the PushTopic message
// that the adapter needs to figure out the subject for the CloudEvent.
type PushTopicSObject struct {
	Name string `json:"entityName"`
}

// EventDispatcher is an object that can dispatch messages and "non managed" errors
// received through the stream.
type EventDispatcher interface {
	DispatchEvent(context.Context, *ConnectResponse)
	DispatchError(error)
}

// Subscription contains data about the channel and replayID for the subscription.
// For replayID can be:
//
//	-2 for all past (stored) and new events.
//	-1 for new events only.
//	replayID for receiving events after that replayID event.
//
// See: https://developer.salesforce.com/docs/atlas.en-us.api_streaming.meta/api_streaming/using_streaming_api_durability.htm
type Subscription struct {
	Channel  string
	ReplayID int
}
