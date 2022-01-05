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
	"encoding/json"
	"net/url"
	"path"

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
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	"github.com/triggermesh/triggermesh/test/e2e/framework/aws"
	e2ekinesis "github.com/triggermesh/triggermesh/test/e2e/framework/aws/kinesis"
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
	targetKind     = "AWSKinesisTarget"
	targetResource = "awskinesistargets"
)

const awsAccessKeyIDSecretKey = "awsApiKey"
const awsSecretAccessKeySecretKey = "awsApiSecret"

var _ = Describe("AWS Kinesis target", func() {
	f := framework.New("awskinesistarget")

	var ns string

	var trgtClient dynamic.ResourceInterface

	var streamARN string
	var streamName string
	var awsSecret *corev1.Secret

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := targetAPIVersion.WithResource(targetResource)
		trgtClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a target is deployed", func() {
		var trgtURL *url.URL
		var kc kinesisiface.KinesisAPI

		BeforeEach(func() {
			sess := session.Must(session.NewSession())
			kc = kinesis.New(sess)

			awsCreds := readAWSCredentials(sess)
			awsSecret = createAWSCredsSecret(f.KubeClient, ns, awsCreds)

			By("creating a Kinesis data stream", func() {
				streamARN = e2ekinesis.CreateDatastream(kc, f)
				streamName = path.Base(aws.ParseARN(streamARN).Resource)

				DeferCleanup(func() {
					By("deleting Kinesis data stream "+streamName, func() {
						e2ekinesis.DeleteStream(kc, streamName)
					})
				})
			})
		})

		When("the spec contains default settings", func() {
			BeforeEach(func() {
				By("creating an AWSKinesisTarget object", func() {
					trgt, err := createTarget(trgtClient, ns, "test-",
						withARN(streamARN),
						withPartition("test"),
						withCredentials(awsSecret.Name),
					)
					Expect(err).ToNot(HaveOccurred())

					trgt = ducktypes.WaitUntilReady(f.DynamicClient, trgt)

					trgtURL = ducktypes.Address(trgt)
					Expect(trgtURL).ToNot(BeNil())
				})
			})

			When("an event is sent to the target", func() {
				var sentEvent *cloudevents.Event

				BeforeEach(func() {
					By("sending an event", func() {
						sentEvent = e2ece.NewHelloEvent(f)

						job := e2ece.RunEventSender(f.KubeClient, ns, trgtURL.String(), sentEvent)
						apps.WaitForCompletion(f.KubeClient, job)
					})
				})

				It("puts a record onto the stream", func() {
					var receivedRcd []byte

					By("polling the stream records", func() {
						receivedRcds := e2ekinesis.GetRecords(kc, streamName)
						Expect(receivedRcds).To(HaveLen(1),
							"Received %d records instead of 1", len(receivedRcds))

						receivedRcd = []byte(receivedRcds[0].Data)
					})

					By("inspecting the record payload", func() {
						RcdData := make(map[string]interface{})
						err := json.Unmarshal(receivedRcd, &RcdData)
						Expect(err).ToNot(HaveOccurred())

						eventData, err := json.Marshal(RcdData)
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
		})

		When("the CloudEvent context is discarded", func() {
			BeforeEach(func() {
				By("creating an AWSKinesisTarget object with discardCEContext enabled", func() {
					trgt, err := createTarget(trgtClient, ns, "test-",
						withARN(streamARN),
						withPartition("test"),
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
				var sentEvent *cloudevents.Event

				BeforeEach(func() {
					By("sending an event", func() {
						sentEvent = e2ece.NewHelloEvent(f)

						job := e2ece.RunEventSender(f.KubeClient, ns, trgtURL.String(), sentEvent)
						apps.WaitForCompletion(f.KubeClient, job)
					})
				})

				It("only puts the event's data onto the record", func() {
					var receivedRcd []byte

					By("polling the stream records", func() {
						receivedRcds := e2ekinesis.GetRecords(kc, streamName)
						Expect(receivedRcds).To(HaveLen(1),
							"Received %d records instead of 1", len(receivedRcds))

						receivedRcd = []byte(receivedRcds[0].Data)
					})
					By("inspecting the record payload", func() {
						Expect(receivedRcd).To(Equal(sentEvent.Data()))
					})
				})
			})
		})
	})

	When("a client creates a target object with invalid specs", func() {

		// Those tests do not require a real streamARN or awsSecret
		BeforeEach(func() {
			streamARN = "arn:aws:kinesis:eu-central-1:000000000000/test"
			awsSecret = &corev1.Secret{}
		})

		// Here we use
		//   "Specify: the API server rejects ..., By: setting an invalid ..."
		// instead of
		//   "When: it sets an invalid ..., Specify: the API server rejects ..."
		// to avoid creating a namespace for each spec, due to their simplicity.
		Specify("the API server rejects the creation of that object", func() {

			By("setting an invalid stream ARN", func() {
				invalidStreamARN := "arn:aws:kinesis:eu-central-1::"

				_, err := createTarget(trgtClient, ns, "test-invalid-arn",
					withARN(invalidStreamARN),
					withPartition("test"),
					withCredentials(awsSecret.Name),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.arn: Invalid value: "))
			})

			By("omitting the stream ARN", func() {
				_, err := createTarget(trgtClient, ns, "test-no-arn",
					withPartition("test"),
					withCredentials(awsSecret.Name),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.arn: Required value"))
			})

			By("omitting the credentials", func() {
				_, err := createTarget(trgtClient, ns, "test-nocreds",
					withARN(streamARN),
					withPartition("test"),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					"spec.awsApiSecret: Required value, spec.awsApiKey: Required value"))
			})
		})
	})
})

// createTarget creates an AWSKinesis object initialized with the given options.
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

func withPartition(partition string) targetOption {
	return func(trgt *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(trgt.Object, partition, "spec", "partition"); err != nil {
			framework.FailfWithOffset(2, "Failed to set spec.partition field: %s", err)
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
