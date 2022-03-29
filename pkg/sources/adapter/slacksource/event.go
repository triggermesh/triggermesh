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

package slacksource

// SlackEvent contains the event payload
type SlackEvent map[string]interface{}

// Type for the event
func (e SlackEvent) Type() string {
	s, ok := e["type"]
	if !ok {
		return ""
	}
	return s.(string)
}

// SlackEventWrapper contains a common wrapper for all events.
// See https://api.slack.com/types/event for reference.
type SlackEventWrapper struct {
	AdditionalProperties map[string]interface{} `json:"-,omitempty"`

	APIAppID    string     `json:"api_app_id"`
	AuthedUsers []string   `json:"authed_users"`
	Event       SlackEvent `json:"event"`
	EventID     string     `json:"event_id"`
	EventTime   int        `json:"event_time"`
	TeamID      string     `json:"team_id"`
	Token       string     `json:"token"`
	Type        string     `json:"type"`
}

// SlackChallenge contains the handshake challenge for
// the Slack events API.
type SlackChallenge struct {
	Category  string `json:"type"`
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
}

// SlackChallengeResponse is the handshake response
// for a challenge.
type SlackChallengeResponse struct {
	Challenge string `json:"challenge"`
}
