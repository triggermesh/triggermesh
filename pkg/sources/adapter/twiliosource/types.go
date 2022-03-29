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

package twiliosource

// Message represents a Twilio SMS.
type Message struct {
	MessageSid    string `json:"message_sid"`
	SmsStatus     string `json:"sms_status"`
	FromCountry   string `json:"from_country"`
	NumSegments   string `json:"num_segments"`
	ToZip         string `json:"to_zip"`
	NumMeda       string `json:"num_meda"`
	AccountSid    string `json:"account_sid"`
	SmsMessageSid string `json:"sms_message_sid"`
	APIVersion    string `json:"api_version"`
	ToCountry     string `json:"to_country"`
	ToCity        string `json:"to_city"`
	FromZip       string `json:"from_zip"`
	SmsSid        string `json:"sms_sid"`
	FromState     string `json:"from_state"`
	Body          string `json:"body"`
	From          string `json:"from"`
	FromCity      string `json:"from_city"`
	To            string `json:"to"`
	ToState       string `json:"to_state"`
}
