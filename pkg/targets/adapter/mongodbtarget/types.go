/*
Copyright 2023 TriggerMesh Inc.

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

package mongodbtarget

import "encoding/json"

// InsertPayload defines the expected data structure found at the "io.triggermesh.mongodb.insert" payload.
type InsertPayload struct {
	Database   string          `json:"database"`
	Collection string          `json:"collection"`
	Key        string          `json:"key"`
	Document   json.RawMessage `json:"document"`
}

// QueryPayload defines the expected data found at the "io.triggermesh.mongodb.query" payload.
type QueryPayload struct {
	Database   string `json:"database"`
	Collection string `json:"collection"`
	Key        string `json:"key"`
	Value      string `json:"value"`
}

// UpdatePayload defines the expected data found at the "io.triggermesh.mongodb.update" payload.
type UpdatePayload struct {
	Database    string `json:"database"`
	Collection  string `json:"collection"`
	SearchKey   string `json:"searchKey"`
	SearchValue string `json:"searchValue"`
	UpdateKey   string `json:"updateKey"`
	UpdateValue string `json:"updateValue"`
}

// QueryResponse defines the expected data structure received from a query to MongoDB.
type QueryResponse struct {
	Collection map[string]string `json:"collection"`
}
