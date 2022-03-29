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

package zendesktarget

// TicketTag is a struct for a ticket tag update payload
type TicketTag struct {
	Tag string `json:"tag,omitempty"`
	ID  int64  `json:"id,omitempty"`
}

// TicketComment is a struct for ticket comment payload
// Via and Metadata are currently unused
// https://developer.zendesk.com/rest_api/docs/support/ticket_comments
type TicketComment struct {
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body,omitempty"`
}
