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

package router

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const headerHandlerName = "HANDLER"

func TestRouter(t *testing.T) {
	r := &Router{}

	assert.Equal(t, 0, r.HandlersCount())

	// new router responds with status NotFound

	resp := recordResponse(t, r, "/")
	assert.Equal(t, http.StatusNotFound, resp.Code)

	// register 2 handlers

	r.RegisterPath("/foo", responder("foo"))
	r.RegisterPath("/bar", responder("bar"))

	assert.Equal(t, 2, r.HandlersCount())

	assert.Equal(t, []string{"/bar", "/foo"}, handlersKeys(r))

	resp = recordResponse(t, r, "/foo")
	assert.Equal(t, "foo", resp.Header().Get(headerHandlerName))

	resp = recordResponse(t, r, "/bar")
	assert.Equal(t, "bar", resp.Header().Get(headerHandlerName))

	// attempt to delete unregistered paths

	r.DeregisterPath("/")
	r.DeregisterPath("/baz")

	assert.Equal(t, 2, r.HandlersCount())

	assert.Equal(t, []string{"/bar", "/foo"}, handlersKeys(r))

	// delete a registered path

	r.DeregisterPath("/foo")

	assert.Equal(t, 1, r.HandlersCount())

	assert.Equal(t, []string{"/bar"}, handlersKeys(r))

	resp = recordResponse(t, r, "/foo")
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

// responder returns a HTTP handler that responds to requests with a header
// containing the given handler's name.
func responder(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(headerHandlerName, name)
		w.WriteHeader(http.StatusNoContent)
	})
}

// handlersKeys returns the keys of all the handlers currently registered in
// the given Router, sorted lexically.
func handlersKeys(r *Router) []string {
	var keys []string

	r.handlers.Range(func(key, _ interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})

	sort.Strings(keys)

	return keys
}

// recordResponse sends a HTTP request to the provided handler at the given
// URL path and returns the recorded response.
func recordResponse(t *testing.T, h http.Handler, urlPath string) *httptest.ResponseRecorder {
	t.Helper()

	req, err := http.NewRequest(http.MethodHead, urlPath, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	return rr
}
