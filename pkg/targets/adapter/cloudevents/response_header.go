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

package cloudevents

import (
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// ResponseHeaderValue is a function that given an
// incoming CloudEvent returns a string to be used
// as a header value on the outgoing event.
type ResponseHeaderValue func(event *cloudevents.Event) (string, error)

// StaticResponse always returns the same fixed value.
// Should be used when returned types or sources do not vary
// depending on the incoming event data.
func StaticResponse(value string) ResponseHeaderValue {
	return func(event *cloudevents.Event) (string, error) {
		return value, nil
	}
}

// MappedResponseType returns an static eventType based on the incoming eventType.
// When no type is mapped a default value is returned.
func MappedResponseType(eventTypes map[string]string) ResponseHeaderValue {
	return func(event *cloudevents.Event) (string, error) {
		v, ok := eventTypes[event.Type()]
		if ok {
			return v, nil
		}
		return "", fmt.Errorf("incoming type %q cannot be mapped to an outgoing event type", event.Type())
	}
}

// SuffixResponseType appends a string to the the incoming eventType.
func SuffixResponseType(suffix string) ResponseHeaderValue {
	return func(event *cloudevents.Event) (string, error) {
		return event.Type() + suffix, nil
	}
}
