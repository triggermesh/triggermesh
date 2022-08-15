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

package googlecloudpubsubsource

import (
	"strings"

	"cloud.google.com/go/pubsub"
)

const ceExtensionPubSubMessagePrefix = "pubsubmsg"

// ceExtensionAttrsForMessage returns a collection of CloudEvents extension
// attributes translated from the message attributes of the given Pub/Sub message.
//
// The resulting extension attribute name is composed of the 'pubsubmsg' prefix,
// followed by the lowercase name of the Pub/Sub message attribute, from which all
// non-alphanumeric characters have been removed (e.g. "pubsubmsgmyattribute").
//
// https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#extension-context-attributes
func ceExtensionAttrsForMessage(msg *pubsub.Message) map[string]interface{} {
	if len(msg.Attributes) == 0 {
		return nil
	}

	ceExtAttrs := make(map[string]interface{})

	for name, attrVal := range msg.Attributes {
		ceExtAttrs[ceExtensionAttrForMessageAttr(name)] = attrVal
	}

	return ceExtAttrs
}

// ceExtensionAttrForMessageAttr sanitizes the name of a Pub/Sub message
// attribute so that it can be used as a CloudEvent context attribute.
//
// The naming conventions for CloudEvent context attributes is described at
// https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#context-attributes.
func ceExtensionAttrForMessageAttr(attrName string) string {
	return ceExtensionPubSubMessagePrefix + stripNonAlphanumCharsAndMapToLower(attrName)
}

// stripNonAlphanumCharsAndMapToLower applies the following transformations to
// the given string:
//   - strips all non alphanumeric characters
//   - maps all Unicode letters to their lower case
func stripNonAlphanumCharsAndMapToLower(s string) string {
	var stripped strings.Builder
	stripped.Grow(len(s))

	// operate on bytes instead of runes, since all alphanumeric characters
	// are represented in a single byte
	for i := 0; i < len(s); i++ {
		b := s[i]

		if ('a' <= b && b <= 'z') ||
			('A' <= b && b <= 'Z') ||
			('0' <= b && b <= '9') {

			if 'A' <= b && b <= 'Z' {
				// shift from upper to lower case
				b += 'a' - 'A'
			}

			stripped.WriteByte(b)
		}
	}

	return stripped.String()
}
