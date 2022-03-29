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

// Package s3 contains helpers for AWS S3.
package s3

import "github.com/triggermesh/triggermesh/pkg/apis"

// RealBucketARN returns a string representation of the given S3 bucket ARN
// which matches the official format defined by AWS.
// https://docs.aws.amazon.com/service-authorization/latest/reference/list_amazons3.html#amazons3-resources-for-iam-policies
//
// This is necessary because our AWSS3Source API accepts that bucket ARNs
// include a region and an account ID, which are both absent from the public
// ARN.
func RealBucketARN(arn apis.ARN) string {
	arn.Region = ""
	arn.AccountID = ""

	return arn.String()
}
