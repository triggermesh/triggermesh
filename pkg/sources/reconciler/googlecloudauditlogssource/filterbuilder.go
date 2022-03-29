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

package googlecloudauditlogssource

import (
	"fmt"
	"strings"
)

const (
	keyPrefix   = "protoPayload"
	methodKey   = keyPrefix + ".methodName"
	serviceKey  = keyPrefix + ".serviceName"
	resourceKey = keyPrefix + ".resourceName"
	typeKey     = keyPrefix + ".\x22@type\x22"
	typeValue   = "type.googleapis.com/google.cloud.audit.AuditLog"
)

// FilterBuilder builds a filter for Google Cloud Audit Logs. Currently
// supports querying by the Audit Logs serviceName, methodName (requireds), and
// resourceName (optional).
type FilterBuilder struct {
	serviceName  string
	methodName   string
	resourceName string
}

// FilterOption is used to apply custom options on FilterBuilder.
type FilterOption func(*FilterBuilder)

// NewFilterBuilder creates a new instance of FilterBuilder.
func NewFilterBuilder(serviceName, methodName string, opts ...FilterOption) *FilterBuilder {
	fb := &FilterBuilder{
		serviceName: serviceName,
		methodName:  methodName,
	}

	for _, f := range opts {
		f(fb)
	}

	return fb
}

// WithResourceName sets FilterBuilder resource name.
func WithResourceName(resourceName string) FilterOption {
	return func(fb *FilterBuilder) {
		fb.resourceName = resourceName
	}
}

// GetFilter returns filter query string.
func (fb *FilterBuilder) GetFilter() string {
	var filters []string

	filters = append(filters, filter{methodKey, fb.methodName}.String())
	filters = append(filters, filter{serviceKey, fb.serviceName}.String())

	if fb.resourceName != "" {
		filters = append(filters, filter{resourceKey, fb.resourceName}.String())
	}

	filters = append(filters, filter{typeKey, typeValue}.String())
	filter := strings.Join(filters, " AND ")
	return filter
}

type filter struct {
	key   string
	value string
}

func (f filter) String() string {
	return fmt.Sprintf("%s=%q", f.key, f.value)
}
