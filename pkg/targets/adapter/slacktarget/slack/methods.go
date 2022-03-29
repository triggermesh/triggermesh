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
	"net/http"
)

// Methods collection for the Slack Web API.
// As listed at https://api.slack.com/web#methods
type Methods map[string]*method

// clientFunc is a function that given a CloudEvent data for a
// Slack supported API performs the required action.
type clientFunc func(data []byte, apiURL, token, method string, client *http.Client) (Response, error)

type method struct {
	enabled bool

	fn clientFunc
}

// GetFullCatalog returns the collection of all supported methods.
func GetFullCatalog(enabled bool) Methods {
	methods := Methods{
		"chat.postMessage":     &method{fn: doJSONPost},
		"chat.scheduleMessage": &method{fn: doJSONPost},
		"chat.update":          &method{fn: doJSONPost},
	}

	for k := range methods {
		methods[k].enabled = enabled
	}
	return methods
}

// Response from Slack Web API
type Response map[string]interface{}

// IsOK returns whether the request was successful
func (r *Response) IsOK() bool {
	v, exist := (*r)["ok"]
	if !exist {
		return false
	}

	isok, ok := v.(bool)
	if !ok {
		return false
	}
	return isok
}

// Error returns the error element if it exists
func (r *Response) Error() string {
	if v, exist := (*r)["error"]; exist {
		return v.(string)
	}
	return ""
}

// Warning returns the warning element if it exists
func (r *Response) Warning() string {
	if v, exist := (*r)["warning"]; exist {
		return v.(string)
	}
	return ""
}

// StatusCode returns the status code for the response
func (r *Response) StatusCode() int {
	if v, exist := (*r)["status"]; exist {
		return v.(int)
	}
	return 0
}
