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

package awss3target

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/arn"
)

// MustParseARN parses an ARN and panics in case of error.
func MustParseARN(arnStr string) arn.ARN {
	arn, err := arn.Parse(arnStr)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %q: %s", arnStr, err))
	}
	return arn
}
