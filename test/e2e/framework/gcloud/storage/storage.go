/*
Copyright 2021 TriggerMesh Inc.

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

// Package storage contains helpers for Google Cloud Storage.
package storage

import (
	"context"

	"cloud.google.com/go/storage"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/gcloud"
)

// CreateBucket creates a bucket named after the given framework.Framework.
func CreateBucket(storageCli *storage.Client, project string, f *framework.Framework) string /*bucket ID*/ {
	bucketID := f.UniqueName

	bucketAttrs := &storage.BucketAttrs{
		// Force single-region by setting an explicit region.
		Location: "us-east1",
		Labels:   gcloud.TagsFor(f),
	}

	if err := storageCli.Bucket(bucketID).Create(context.Background(), project, bucketAttrs); err != nil {
		framework.FailfWithOffset(2, "Failed to create bucket %q: %s", bucketID, err)
	}

	return bucketID
}

// CreateObject creates an object in the given bucket.
func CreateObject(storageCli *storage.Client, bucket string, f *framework.Framework) string /*obj name*/ {
	const objectName = "hello.txt"

	objWriter := storageCli.Bucket(bucket).Object(objectName).NewWriter(context.Background())
	defer func() {
		if err := objWriter.Close(); err != nil {
			framework.FailfWithOffset(2, "Failed to close writer for object %q: %s", objectName, err)
		}
	}()

	if _, err := objWriter.Write([]byte("Hello, World!")); err != nil {
		framework.FailfWithOffset(2, "Failed to create object %q: %s", objectName, err)
	}

	return objectName
}

// DeleteBucket deletes a bucket by ID.
// Buckets need to be emptied before they can be deleted.
func DeleteBucket(storageCli *storage.Client, bucketID string) {
	if err := storageCli.Bucket(bucketID).Delete(context.Background()); err != nil {
		framework.FailfWithOffset(2, "Failed to delete bucket %q: %s", bucketID, err)
	}
}

// DeleteObject deletes an object by name from the given bucket.
func DeleteObject(storageCli *storage.Client, bucketID, objectName string) {
	if err := storageCli.Bucket(bucketID).Object(objectName).Delete(context.Background()); err != nil {
		framework.FailfWithOffset(2, "Failed to delete object %q from bucket %q: %s",
			objectName, bucketID, err)
	}
}
