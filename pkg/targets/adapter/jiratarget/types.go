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

package jiratarget

import (
	"encoding/json"
	"net/http"

	"github.com/andygrunwald/go-jira"
)

// Method performed on the API URI
type Method string

// Available actions at the Salesforce API
const (
	MethodCreate Method = http.MethodPost
	MethodPut    Method = http.MethodPut
	MethodPatch  Method = http.MethodPatch
	MethodGet    Method = http.MethodGet
	MethodDelete Method = http.MethodDelete
)

// JiraAPIRequest contains common parameters used for
// interacting with Jira using the API.
type JiraAPIRequest struct {
	Method  Method            `json:"method"`
	Path    string            `json:"path"`
	Query   map[string]string `json:"query"`
	Payload json.RawMessage   `json:"payload"`
}

// IssueGetRequest contains parameters for issue retrieval
type IssueGetRequest struct {
	ID      string               `json:"id"`
	Options jira.GetQueryOptions `json:"options"`
}
