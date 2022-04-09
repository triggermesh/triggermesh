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

package awssqssource

import (
	"strings"

	"github.com/aws/aws-sdk-go/service/sqs"
)

const sqsMgsAttrDataTypeBinary = "Binary"
const ceExtensionSQSMessagePrefix = "sqsmsg"

// ceExtensionAttrsForMessage returns a collection of CloudEvents extension
// attributes translated from the message attributes of the given SQS message.
//
// Attributes with a Binary data type are excluded.
// The resulting extension attribute name is composed of the 'sqsmsg' prefix,
// followed by the lowercase name of the SQS message attribute, from which all
// non-alphanumeric characters have been removed (e.g. "sqsmsgmyattribute").
//
// https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#extension-context-attributes
func ceExtensionAttrsForMessage(msg *sqs.Message) map[string]interface{} {
	if len(msg.MessageAttributes) == 0 {
		return nil
	}

	ceExtAttrs := make(map[string]interface{})

	for name, attrVal := range msg.MessageAttributes {
		if !strings.HasPrefix(*attrVal.DataType, sqsMgsAttrDataTypeBinary) {
			ceExtAttrs[ceExtensionAttrForMessageAttr(name)] = *attrVal.StringValue
		}
	}

	return ceExtAttrs
}

// ceExtensionAttrForMessageAttr sanitizes the name of a SQS message attribute
// so that it can be used as a CloudEvent context attribute.
//
// The naming conventions for both formats are described at:
//  - https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-message-metadata.html#message-attribute-components
//  - https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#context-attributes
func ceExtensionAttrForMessageAttr(attrName string) string {
	return ceExtensionSQSMessagePrefix + stripNonAlphanumCharsAndMapToLower(attrName)
}

// stripNonAlphanumCharsAndMapToLower applies the following transformations to
// the given string:
//  - strips all non alphanumeric characters
//  - maps all Unicode letters to their lower case
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
