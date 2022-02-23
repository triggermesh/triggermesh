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

package probe

import "net/http"

// EndpointPath is the recommended URL path of the health endpoint to serve a
// ReadinessChecker over HTTP.
const EndpointPath = "/health"

// ignoreHealthEndpoint allows ignoring the health endpoint in results returned
// by router.HandlersCount().
func ignoreHealthEndpoint(urlPath string) bool {
	return urlPath == EndpointPath
}

// ReadinessCheckerHTTPHandler returns a http.Handler which executes the given ReadinessChecker.
func ReadinessCheckerHTTPHandler(c ReadinessChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isReady, err := c.IsReady()
		switch {
		case err != nil:
			w.WriteHeader(http.StatusInternalServerError)
		case isReady:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
}
