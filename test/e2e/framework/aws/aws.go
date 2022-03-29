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

// Package aws contains helpers to interact with AWS services.
package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

const e2eInstanceTagKey = "e2e_instance"

// ParseARN parses an ARN string into an arn.ARN.
func ParseARN(arnStr string) arn.ARN {
	arn, err := arn.Parse(arnStr)
	if err != nil {
		framework.FailfWithOffset(2, "Error parsing ARN string %q: %s", arnStr, err)
	}

	return arn
}

// TagsFor returns a set of resource tags matching the given framework.Framework.
func TagsFor(f *framework.Framework) map[string]*string {
	return aws.StringMap(map[string]string{
		e2eInstanceTagKey: f.UniqueName,
	})
}
