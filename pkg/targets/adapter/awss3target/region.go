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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Per AWS conventions, a bucket which does not explicitly specify its location
// is located in the "us-east-1" region.
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLocation.html
const defaultS3Region = "us-east-1"

// getBucketRegion retrieves the region the provided bucket resides in.
func getBucketRegion(bucketName string, env *envAccessor) (string, error) {
	sess := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(defaultS3Region)))

	config := &aws.Config{}
	if env.AssumeIamRole != "" {
		config.Credentials = stscreds.NewCredentials(sess, env.AssumeIamRole)
	}

	resp, err := s3.New(sess, config).GetBucketLocation(&s3.GetBucketLocationInput{
		Bucket: &bucketName,
	})
	if err != nil {
		return "", err
	}

	if loc := resp.LocationConstraint; loc != nil {
		return *loc, nil
	}

	return defaultS3Region, nil
}
