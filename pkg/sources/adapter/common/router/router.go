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
	"html"
	"net/http"
	"sync"
)

// Router routes incoming HTTP requests to the adequate handler based on their
// URL path.
type Router struct {
	// map of URL path to HTTP handler
	handlers sync.Map
}

// Check that Router implements http.Handler.
var _ http.Handler = (*Router)(nil)

// RegisterPath registers a HTTP handler for serving requests at the given URL path.
func (r *Router) RegisterPath(urlPath string, h http.Handler) {
	r.handlers.Store(urlPath, h)
}

// DeregisterPath de-registers the HTTP handler for the given URL path.
func (r *Router) DeregisterPath(urlPath string) {
	r.handlers.Delete(urlPath)
}

// HandlersCount returns the number of handlers that are currently registered.
func (r *Router) HandlersCount(filters ...handlerMatcherFunc) int {
	var count int

	r.handlers.Range(func(urlPath, _ interface{}) bool {
		for _, f := range filters {
			if f(urlPath.(string)) {
				return true
			}
		}

		count++

		return true
	})

	return count
}

// handlerMatcherFunc is a matcher that allows ignoring some of the handlers
// inside HandlersCount() based on arbitrary predicates.
// The function should return 'true' if the given urlPath matches the
// predicate, in which case the handler is ignored.
type handlerMatcherFunc func(urlPath string) bool

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h, ok := r.handlers.Load(req.URL.Path)
	if !ok {
		http.Error(w, "No handler for path "+html.EscapeString(req.URL.Path), http.StatusNotFound)
		return
	}

	h.(http.Handler).ServeHTTP(w, req)
}
