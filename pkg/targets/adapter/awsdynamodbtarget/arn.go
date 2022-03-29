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

package awsdynamodbtarget

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
)

const (
	expectDynamoDBResourceFmt = "table/${TableName}"
)

// MustParseARN parses an ARN and panics in case of error.
func MustParseARN(arnStr string) arn.ARN {
	arn, err := arn.Parse(arnStr)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %q: %s", arnStr, err))
	}
	return arn
}

// MustParseDynamoDBResource parses the resource segment of a DynamoDB ARN and
// panics in case of error.
func MustParseDynamoDBResource(resource string) string /*table*/ {
	elements, err := parseResource(resource, expectDynamoDBResourceFmt)
	if err != nil {
		panic(err)
	}
	return elements[0]
}

// parseResource parses the resource segment of a ARN and panics in case of
// error. A ARN resource should have the format "key1/val1/key2/val2/...".
func parseResource(resource, expectFormat string) ([]string, error) {
	expectElements := strings.Split(expectFormat, "/")

	sections := strings.Split(resource, "/")
	if len(sections) != len(expectElements) {
		return nil, newParseResourceError(expectFormat, resource)
	}

	// exclude keys, only count values
	elements := make([]string, 0, len(expectElements)/2)

	for i, sec := range sections {
		// assert equality of keys (even indexes), we want them to
		// match the expected format unconditionally
		if i%2 == 0 {
			if sec != expectElements[i] {
				return nil, newParseResourceError(expectFormat, resource)
			}
			continue
		}
		elements = append(elements, sec)
	}

	return elements, nil
}

type parseResourceError struct {
	expectedFormat string
	gotInput       string
}

func newParseResourceError(expect, got string) error {
	return &parseResourceError{
		expectedFormat: expect,
		gotInput:       got,
	}
}

// Error implements the error interface.
func (e *parseResourceError) Error() string {
	return fmt.Sprintf("resource segment of ARN %q does not match expected format %q", e.gotInput, e.expectedFormat)
}
