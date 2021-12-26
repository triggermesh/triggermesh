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
	"io"
	"os"

	"cloud.google.com/go/storage"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

const (
	defaultStorageClass = "STANDARD"
	defaultLocation     = "EU"
)

// CreateBucket creates a bucket named after the given framework.Framework.
func CreateBucket(storageCli *storage.Client, project string, f *framework.Framework) string {
	storageClassAndLocation := &storage.BucketAttrs{
		StorageClass: defaultStorageClass,
		Location:     defaultLocation,
	}

	bucketName := f.UniqueName

	createBucket := storageCli.Bucket(bucketName)
	if err := createBucket.Create(context.Background(), project, storageClassAndLocation); err != nil {
		framework.FailfWithOffset(2, "Failed to create bucket %q: %s", bucketName, err)
	}

	return bucketName
}

// CreateObject creates an object named after the given framework.Framework.
func CreateObject(storageCli *storage.Client, project string, bucket string, f *framework.Framework) string {
	object := f.UniqueName
	file, err := os.Create("/tmp/" + object)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create file %q: %s", object, err)
	}

	wc := storageCli.Bucket(bucket).Object(object).NewWriter(context.Background())
	if _, err = io.Copy(wc, file); err != nil {
		framework.FailfWithOffset(2, "Failed to create object %q: %s", object, err)
	}
	if err := wc.Close(); err != nil {
		framework.FailfWithOffset(2, "Failed to create object %q: %s", object, err)
	}

	return object
}

// DeleteBucket deletes a bucket.
func DeleteBucket(storageCli *storage.Client, bucketName string) {
	bucket := storageCli.Bucket(bucketName)

	if err := bucket.Delete(context.Background()); err != nil {
		framework.FailfWithOffset(2, "Failed to delete bucket %q: %s", bucketName, err)
	}
}

// DeleteObject deletes an object.
func DeleteObject(storageCli *storage.Client, bucketName string, objectName string) {
	if err := os.Remove("/tmp/" + objectName); err != nil {
		framework.FailfWithOffset(2, "Failed to delete file %q: %s", objectName, err)
	}

	object := storageCli.Bucket(bucketName).Object(objectName)
	if err := object.Delete(context.Background()); err != nil {
		framework.FailfWithOffset(2, "Failed to delete object %q: %s", objectName, err)
	}
}
