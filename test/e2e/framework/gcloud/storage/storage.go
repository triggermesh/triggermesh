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

// Package storage contains helpers for Google Cloud Storage.
package storage

import (
	"context"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/gcloud"
)

// CreateBucket creates a bucket named after the given framework.Framework.
func CreateBucket(storageCli *storage.Client, project string, f *framework.Framework) string /*bucket Name*/ {
	bucketName := f.UniqueName

	bucketAttrs := &storage.BucketAttrs{
		// Force single-region by setting an explicit region.
		Location: "us-east1",
		Labels:   gcloud.TagsFor(f),
	}

	if err := storageCli.Bucket(bucketName).Create(context.Background(), project, bucketAttrs); err != nil {
		framework.FailfWithOffset(2, "Failed to create bucket %q: %s", bucketName, err)
	}

	return bucketName
}

// CreateObject creates an object in the given bucket.
func CreateObject(storageCli *storage.Client, bucketName string, f *framework.Framework) string /*obj name*/ {
	const objectName = "hello.txt"

	objWriter := storageCli.Bucket(bucketName).Object(objectName).NewWriter(context.Background())
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

// DeleteBucket deletes a bucket by name.
func DeleteBucket(storageCli *storage.Client, bucketName string) {
	bucket := storageCli.Bucket(bucketName)
	objects := getObjects(storageCli, bucketName)

	for _, o := range objects {
		err := bucket.Object(o).Delete(context.Background())
		if err != nil {
			framework.FailfWithOffset(2, "Failed to delete objects from bucket %s", err)
		}
	}

	if err := storageCli.Bucket(bucketName).Delete(context.Background()); err != nil {
		framework.FailfWithOffset(2, "Failed to delete bucket %q: %s", bucketName, err)
	}
}

// DeleteObject deletes an object by name from the given bucket.
func DeleteObject(storageCli *storage.Client, bucketName, objectName string) {
	if err := storageCli.Bucket(bucketName).Object(objectName).Delete(context.Background()); err != nil {
		framework.FailfWithOffset(2, "Failed to delete object %q from bucket %q: %s",
			objectName, bucketName, err)
	}
}

// GetObjectsReader gets objects readers from a storage bucket.
func GetObjectsReader(storageCli *storage.Client, bucketName string) []*storage.Reader {
	var objectReaderList []*storage.Reader

	bucket := storageCli.Bucket(bucketName)
	objects := getObjects(storageCli, bucketName)

	for _, o := range objects {
		objectReader, err := bucket.Object(o).NewReader(context.Background())
		if err != nil {
			framework.FailfWithOffset(2, "Failed to get objects reader from bucket %s", err)
		}
		defer objectReader.Close()

		objectReaderList = append(objectReaderList, objectReader)
	}

	return objectReaderList
}

// getObjects gets objects from a storage bucket.
func getObjects(storageCli *storage.Client, bucketName string) []string {
	var objectList []string
	query := &storage.Query{Prefix: ""}

	bucket := storageCli.Bucket(bucketName)
	objects := bucket.Objects(context.Background(), query)

	for {
		attrs, err := objects.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			framework.FailfWithOffset(2, "Failed to get objects from bucket %s", err)
		}

		objectList = append(objectList, attrs.Name)
	}

	return objectList
}
