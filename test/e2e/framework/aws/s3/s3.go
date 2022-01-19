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

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateBucket creates a bucket named after the given framework.Framework.
func CreateBucket(s3Client *s3.S3, f *framework.Framework, region string) string {
	bucketInput := &s3.CreateBucketInput{
		Bucket: &f.UniqueName,
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(region),
		},
	}

	_, err := s3Client.CreateBucket(bucketInput)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create bucket %q: %s", *bucketInput.Bucket, err)
	}

	if err := s3Client.WaitUntilBucketExists(&s3.HeadBucketInput{Bucket: bucketInput.Bucket}); err != nil {
		framework.FailfWithOffset(2, "Failed while waiting for bucket to exist: %s", err)
	}

	return *bucketInput.Bucket
}

// GetObjects get objects from a s3 bucket.
func GetObjects(s3Client *s3.S3, bucketName string) []*s3.GetObjectOutput {
	var objectList []*s3.GetObjectOutput

	objects, err := s3Client.ListObjects(&s3.ListObjectsInput{
		Bucket: &bucketName,
	})
	if err != nil {
		framework.FailfWithOffset(2, "Failed to get objects from bucket: %s", err)
	}

	for _, o := range objects.Contents {
		object, err := s3Client.GetObject(&s3.GetObjectInput{
			Bucket: &bucketName,
			Key:    o.Key,
		})
		if err != nil {
			framework.FailfWithOffset(2, "Failed to get object from bucket: %s", err)
		}

		objectList = append(objectList, object)
	}

	return objectList
}

// DeleteBucket deletes a s3 bucket by name.
func DeleteBucket(s3Client *s3.S3, bucketName string) {
	bucket := &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	iter := s3manager.NewDeleteListIterator(s3Client, &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})

	if err := s3manager.NewBatchDeleteWithClient(s3Client).Delete(aws.BackgroundContext(), iter); err != nil {
		framework.FailfWithOffset(2, "Unable to delete objects from bucket %q: %s", bucketName, err)
	}

	if _, err := s3Client.DeleteBucket(bucket); err != nil {
		framework.FailfWithOffset(2, "Failed to delete bucket %q: %s", bucketName, err)
	}
}
