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

package awss3

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"

	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	e2es3 "github.com/triggermesh/triggermesh/test/e2e/framework/aws/s3"
	e2ece "github.com/triggermesh/triggermesh/test/e2e/framework/cloudevents"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
)

/* This test suite requires:

   - AWS credentials in whichever form (https://docs.aws.amazon.com/sdk-for-go/api/aws/session/#hdr-Sessions_options_from_Shared_Config)
   - The name of an AWS region exported in the environment as AWS_REGION
*/

var targetAPIVersion = schema.GroupVersion{
	Group:   "targets.triggermesh.io",
	Version: "v1alpha1",
}

const (
	targetKind     = "AWSS3Target"
	targetResource = "awss3targets"
)

const awsAccessKeyIDSecretKey = "awsApiKey"
const awsSecretAccessKeySecretKey = "awsApiSecret"

var _ = Describe("AWS S3 target", func() {
	f := framework.New("awss3target")
	region := os.Getenv("AWS_REGION")

	var ns string

	var trgtClient dynamic.ResourceInterface

	var bucketARN string
	var awsSecret *corev1.Secret

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := targetAPIVersion.WithResource(targetResource)
		trgtClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a target is deployed", func() {
		var trgtURL *url.URL
		var s3Client *s3.S3

		var sentEvent *cloudevents.Event

		var bucketName string

		BeforeEach(func() {
			sess := session.Must(session.NewSession())
			s3Client = s3.New(sess)

			awsCreds := readAWSCredentials(sess)
			awsSecret = createAWSCredsSecret(f.KubeClient, ns, awsCreds)

			By("creating a S3 bucket", func() {
				bucketName = e2es3.CreateBucket(s3Client, f, region)
				bucketARN = createBucketARN(bucketName)

				DeferCleanup(func() {
					By("deleting S3 bucket "+bucketName, func() {
						e2es3.DeleteBucket(s3Client, bucketName)
					})
				})
			})
		})

		When("the spec contains default settings", func() {
			BeforeEach(func() {
				By("creating an AWSS3Target object", func() {
					trgt, err := createTarget(trgtClient, ns, "test-",
						withARN(bucketARN),
						withCredentials(awsSecret.Name),
					)
					Expect(err).ToNot(HaveOccurred())

					trgt = ducktypes.WaitUntilReady(f.DynamicClient, trgt)

					trgtURL = ducktypes.Address(trgt)
					Expect(trgtURL).ToNot(BeNil())
				})
			})

			When("an event is sent to the target", func() {

				BeforeEach(func() {
					By("sending an event", func() {
						sentEvent = e2ece.NewHelloEvent(f)

						job := e2ece.RunEventSender(f.KubeClient, ns, trgtURL.String(), sentEvent)
						apps.WaitForCompletion(f.KubeClient, job)
					})
				})

				It("creates an object onto the bucket", func() {
					var receivedObj []byte
					var err error

					By("listing the bucket objects", func() {
						receivedObjs := e2es3.GetObjects(s3Client, bucketName)
						Expect(receivedObjs).To(HaveLen(1),
							"Received %d objects instead of 1", len(receivedObjs))

						object := *receivedObjs[0]
						receivedObj, err = ioutil.ReadAll(object.Body)
						Expect(err).ToNot(HaveOccurred())
					})

					By("inspecting the object payload", func() {
						ObjData := make(map[string]interface{})
						err := json.Unmarshal(receivedObj, &ObjData)
						Expect(err).ToNot(HaveOccurred())

						eventData, err := json.Marshal(ObjData)
						Expect(err).ToNot(HaveOccurred())

						gotEvent := &cloudevents.Event{}
						err = json.Unmarshal(eventData, gotEvent)
						Expect(err).ToNot(HaveOccurred())

						Expect(gotEvent.ID()).To(Equal(sentEvent.ID()))
						Expect(gotEvent.Type()).To(Equal(sentEvent.Type()))
						Expect(gotEvent.Source()).To(Equal(sentEvent.Source()))
						Expect(gotEvent.Data()).To(Equal(sentEvent.Data()))
						Expect(gotEvent.Extensions()[e2ece.E2ECeExtension]).
							To(Equal(sentEvent.Extensions()[e2ece.E2ECeExtension]))
					})
				})
			})
			When("an event with specific type is sent to the target", func() {

				BeforeEach(func() {
					By("sending an event with type io.triggermesh.awss3.object.put", func() {
						sentEvent = e2ece.NewHelloEvent(f)
						sentEvent.SetType("io.triggermesh.awss3.object.put")

						job := e2ece.RunEventSender(f.KubeClient, ns, trgtURL.String(), sentEvent)
						apps.WaitForCompletion(f.KubeClient, job)
					})
				})

				It("only puts the event's data onto the bucket object", func() {
					var receivedObj []byte
					var err error

					By("listing the bucket objects", func() {
						receivedObjs := e2es3.GetObjects(s3Client, bucketName)
						Expect(receivedObjs).To(HaveLen(1),
							"Received %d objects instead of 1", len(receivedObjs))

						object := *receivedObjs[0]
						receivedObj, err = ioutil.ReadAll(object.Body)
						Expect(err).ToNot(HaveOccurred())
					})

					By("inspecting the object payload", func() {
						Expect(receivedObj).To(Equal(sentEvent.Data()))
					})
				})
			})
		})

		When("the CloudEvent context is discarded", func() {
			BeforeEach(func() {
				By("creating an AWSS3Target object with discardCEContext enabled", func() {
					trgt, err := createTarget(trgtClient, ns, "test-",
						withARN(bucketARN),
						withCredentials(awsSecret.Name),
						withDiscardCEContext(),
					)
					Expect(err).ToNot(HaveOccurred())

					trgt = ducktypes.WaitUntilReady(f.DynamicClient, trgt)

					trgtURL = ducktypes.Address(trgt)
					Expect(trgtURL).ToNot(BeNil())
				})
			})
			When("an event is sent to the target", func() {
				BeforeEach(func() {
					By("sending an event", func() {
						sentEvent = e2ece.NewHelloEvent(f)

						job := e2ece.RunEventSender(f.KubeClient, ns, trgtURL.String(), sentEvent)
						apps.WaitForCompletion(f.KubeClient, job)
					})
				})

				It("only puts the event's data onto the bucket object", func() {
					var receivedObj []byte
					var err error

					By("listing the bucket objects", func() {
						receivedObjs := e2es3.GetObjects(s3Client, bucketName)
						Expect(receivedObjs).To(HaveLen(1),
							"Received %d objects instead of 1", len(receivedObjs))

						object := *receivedObjs[0]
						receivedObj, err = ioutil.ReadAll(object.Body)
						Expect(err).ToNot(HaveOccurred())
					})
					By("inspecting the object payload", func() {
						Expect(receivedObj).To(Equal(sentEvent.Data()))
					})
				})
			})
		})
	})

	When("a client creates a target object with invalid specs", func() {

		// Those tests do not require a real bucketARN or awsSecret
		BeforeEach(func() {
			bucketARN = "arn:aws:s3:::test"
			awsSecret = &corev1.Secret{}
		})

		// Here we use
		//   "Specify: the API server rejects ..., By: setting an invalid ..."
		// instead of
		//   "When: it sets an invalid ..., Specify: the API server rejects ..."
		// to avoid creating a namespace for each spec, due to their simplicity.
		Specify("the API server rejects the creation of that object", func() {

			By("setting an invalid bucket ARN", func() {
				invalidBucketARN := "arn:aws:s3:::"

				_, err := createTarget(trgtClient, ns, "test-invalid-arn",
					withARN(invalidBucketARN),
					withCredentials(awsSecret.Name),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.arn: Invalid value: "))
			})

			By("omitting the bucket ARN", func() {
				_, err := createTarget(trgtClient, ns, "test-no-arn",
					withCredentials(awsSecret.Name),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.arn: Required value"))
			})

			By("omitting the credentials", func() {
				_, err := createTarget(trgtClient, ns, "test-nocreds",
					withARN(bucketARN),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					"spec.awsApiSecret: Required value, spec.awsApiKey: Required value"))
			})
		})
	})
})

// createTarget creates an AWSS3Target object initialized with the given options.
func createTarget(trgtClient dynamic.ResourceInterface, namespace, namePrefix string, opts ...targetOption) (*unstructured.Unstructured, error) {

	trgt := &unstructured.Unstructured{}
	trgt.SetAPIVersion(targetAPIVersion.String())
	trgt.SetKind(targetKind)
	trgt.SetNamespace(namespace)
	trgt.SetGenerateName(namePrefix)

	for _, opt := range opts {
		opt(trgt)
	}

	return trgtClient.Create(context.Background(), trgt, metav1.CreateOptions{})
}

type targetOption func(*unstructured.Unstructured)

func withARN(arn string) targetOption {
	return func(trgt *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(trgt.Object, arn, "spec", "arn"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.arn field: %s", err)
		}
	}
}

func withDiscardCEContext() targetOption {
	return func(trgt *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(trgt.Object, true, "spec", "discardCloudEventContext"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.discardCloudEventContext field: %s", err)
		}
	}
}

func withCredentials(secretName string) targetOption {
	apiKeySecretRef := map[string]interface{}{
		"secretKeyRef": map[string]interface{}{
			"name": secretName,
			"key":  awsAccessKeyIDSecretKey,
		},
	}

	apiSecretSecretRef := map[string]interface{}{
		"secretKeyRef": map[string]interface{}{
			"name": secretName,
			"key":  awsSecretAccessKeySecretKey,
		},
	}

	return func(trgt *unstructured.Unstructured) {
		if err := unstructured.SetNestedMap(trgt.Object, apiKeySecretRef, "spec", awsAccessKeyIDSecretKey); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.accessToken field: %s", err)
		}
		if err := unstructured.SetNestedMap(trgt.Object, apiSecretSecretRef, "spec", awsSecretAccessKeySecretKey); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.secretToken field: %s", err)
		}
	}
}

// createAWSCredsSecret creates a Kubernetes Secret containing a AWS credentials.
func createAWSCredsSecret(c clientset.Interface, namespace string, creds credentials.Value) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: "aws-creds-",
		},
		StringData: map[string]string{
			awsAccessKeyIDSecretKey:     creds.AccessKeyID,
			awsSecretAccessKeySecretKey: creds.SecretAccessKey,
		},
	}

	var err error

	secret, err = c.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		framework.FailfWithOffset(2, "Failed to create Secret: %s", err)
	}

	return secret
}

func readAWSCredentials(sess *session.Session) credentials.Value {
	creds, err := sess.Config.Credentials.Get()
	if err != nil {
		framework.FailfWithOffset(2, "Error reading AWS credentials: %s", err)
	}

	return creds
}

// createBucketARN will create the bucket ARN used by the k8s awss3target
func createBucketARN(bucketName string) string {
	return "arn:aws:s3:::" + bucketName
}
