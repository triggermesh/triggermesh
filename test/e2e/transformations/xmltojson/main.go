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

package awskinesis

import (
	"context"

	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

var targetAPIVersion = schema.GroupVersion{
	Group:   "flow.triggermesh.io",
	Version: "v1alpha1",
}

const (
	transformationKind     = "XMLToJSONTransformation"
	transformationResource = "xmltojsontransformations"
)

var _ = Describe("AWS Kinesis target", func() {
	f := framework.New("xmltojsontransformation")

	var ns string

	var trgtClient dynamic.ResourceInterface

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := targetAPIVersion.WithResource(transformationResource)
		trgtClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Specify("the API server rejects the creation of that object", func() {

		By("setting an invalid stream ARN", func() {
			_, err := createTransformation(trgtClient, ns, "test-invalid-arn")
			Expect(err).ToNot(HaveOccurred())
		})
	})

})

// createTransformation creates an AWSKinesis object initialized with the given options.
func createTransformation(trgtClient dynamic.ResourceInterface, namespace, namePrefix string, opts ...targetOption) (*unstructured.Unstructured, error) {

	trgt := &unstructured.Unstructured{}
	trgt.SetAPIVersion(targetAPIVersion.String())
	trgt.SetKind(transformationKind)
	trgt.SetNamespace(namespace)
	trgt.SetGenerateName(namePrefix)

	for _, opt := range opts {
		opt(trgt)
	}

	return trgtClient.Create(context.Background(), trgt, metav1.CreateOptions{})
}

type targetOption func(*unstructured.Unstructured)
