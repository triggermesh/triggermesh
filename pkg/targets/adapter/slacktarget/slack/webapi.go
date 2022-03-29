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

package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// WebAPIClient is an HTTP client for Slack Web API.
type WebAPIClient interface {
	Do(methodURL string, body []byte) (Response, error)
}

// NewWebAPIClient returns the default implementation of the Slack Web API client.
func NewWebAPIClient(token, apiURL string, client *http.Client, methods Methods) WebAPIClient {
	return &webAPIClient{
		client:  client,
		apiURL:  apiURL,
		token:   token,
		methods: methods,
	}
}

type webAPIClient struct {
	client  *http.Client
	methods Methods
	apiURL  string
	token   string
}

// Do looks for the method at the catalog and if found and enabled executes the request.
func (c *webAPIClient) Do(methodURL string, data []byte) (Response, error) {
	method, ok := c.methods[methodURL]
	if !ok {
		return nil, fmt.Errorf("Slack method %q not supported", methodURL) //nolint:stylecheck
	}

	if !method.enabled {
		return nil, fmt.Errorf("Slack method %q is not enabled", methodURL) //nolint:stylecheck
	}

	// this should not happen
	if method.fn == nil {
		return nil, fmt.Errorf("Slack method %q is not implemented", methodURL) //nolint:stylecheck
	}

	return method.fn(data, c.apiURL, c.token, methodURL, c.client)
}

// doJSONPost is the API processing function for requests that POST its data as JSON
func doJSONPost(data []byte, apiURL, token, method string, client *http.Client) (Response, error) {
	req, err := http.NewRequest("POST", apiURL+method, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	r := make(Response)
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		// create a response based on the decoding error
		r["ok"] = "false"
		r["error"] = err.Error()
	}

	r["status"] = res.StatusCode

	return r, nil
}
