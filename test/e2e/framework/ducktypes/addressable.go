/*
Copyright 2020 TriggerMesh Inc.

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

package ducktypes

import (
	"net/url"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// Address returns the address of an Addressable object as a URL.
// Fails the test if not found.
func Address(obj *unstructured.Unstructured) *url.URL {
	return (*url.URL)(unstructuredToAddressableType(obj).Status.Address.URL)
}

func unstructuredToAddressableType(obj *unstructured.Unstructured) *duckv1.AddressableType {
	a := &duckv1.AddressableType{}
	if err := duck.FromUnstructured(obj, a); err != nil {
		framework.FailfWithOffset(2, "Failed to convert unstructured object to Addressable: %s", err)
	}
	return a
}
