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

package awscognitouserpool

import (
	"context"
	"path"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:stylecheck
	. "github.com/onsi/gomega"    //nolint:stylecheck

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"

	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
	"github.com/triggermesh/triggermesh/test/e2e/framework/apps"
	"github.com/triggermesh/triggermesh/test/e2e/framework/aws"
	e2ecognitouserpool "github.com/triggermesh/triggermesh/test/e2e/framework/aws/cognitouserpool"
	"github.com/triggermesh/triggermesh/test/e2e/framework/bridges"
	"github.com/triggermesh/triggermesh/test/e2e/framework/ducktypes"
)

/* This test suite requires:

   - AWS credentials in whichever form (https://docs.aws.amazon.com/sdk-for-go/api/aws/session/#hdr-Sessions_options_from_Shared_Config)
   - The name of an AWS region exported in the environment as AWS_REGION
*/

var sourceAPIVersion = schema.GroupVersion{
	Group:   "sources.triggermesh.io",
	Version: "v1alpha1",
}

const (
	sourceKind     = "AWSCognitoUserPoolSource"
	sourceResource = "awscognitouserpoolsources"
)

var _ = Describe("AWS Cognito UserPool source", func() {
	f := framework.New("awscognitouserpoolsource")

	var ns string

	var srcClient dynamic.ResourceInterface

	var poolARN string

	var awsCreds credentials.Value
	var sink *duckv1.Destination

	BeforeEach(func() {
		ns = f.UniqueName

		gvr := sourceAPIVersion.WithResource(sourceResource)
		srcClient = f.DynamicClient.Resource(gvr).Namespace(ns)
	})

	Context("a source watches an existing userpool", func() {
		var cc cognitoidentityprovideriface.CognitoIdentityProviderAPI
		var poolID string

		BeforeEach(func() {
			sess := session.Must(session.NewSession())
			cc = cognitoidentityprovider.New(sess)
			awsCreds = readAWSCredentials(sess)

			By("creating an event sink", func() {
				sink = bridges.CreateEventDisplaySink(f.KubeClient, ns)
			})

			By("creating a Cognito userpool", func() {
				poolARN = e2ecognitouserpool.CreateUserPool(cc, f)
				poolID = path.Base(aws.ParseARN(poolARN).Resource)

				// TODO: remove after https://github.com/triggermesh/triggermesh/issues/35 is resolved
				e2ecognitouserpool.CreateUser(cc, poolID, "alice")
			})

			By("creating an AWSCognitoUserPoolSource object", func() {
				src, err := createSource(srcClient, ns, "test-", sink,
					withARN(poolARN),
					withCredentials(awsCreds),
				)
				Expect(err).ToNot(HaveOccurred())

				ducktypes.WaitUntilReady(f.DynamicClient, src)

				// FIXME(antoineco): without this short pause, the receive adapter throws the following
				// error when sending the event:
				//
				//   Sending Cognito event
				//   Post "http://event-display.{...}": dial tcp 10.x.x.x:80: connect: connection refused
				//
				time.Sleep(2 * time.Second)
			})
		})

		AfterEach(func() {
			By("deleting Cognito userpool "+poolID, func() {
				e2ecognitouserpool.DeleteUserPool(cc, poolID)
			})
		})

		When("a user is created in the userpool", func() {
			const username = "bob"

			BeforeEach(func() {
				e2ecognitouserpool.CreateUser(cc, poolID, username)
			})

			Specify("the source generates an event for created user", func() {
				const receiveTimeout = 10 * time.Second
				const pollInterval = 500 * time.Millisecond

				var receivedEvents []cloudevents.Event

				readReceivedEvents := readReceivedEvents(f.KubeClient, ns, sink.Ref.Name, &receivedEvents)

				Eventually(readReceivedEvents, receiveTimeout, pollInterval).ShouldNot(BeEmpty())
				Expect(receivedEvents).To(HaveLen(1))

				e := receivedEvents[0]

				var user cognitoidentityprovider.UserType

				err := e.DataAs(&user)
				Expect(err).NotTo(HaveOccurred())

				Expect(e.Type()).To(Equal("com.amazon.cognito-idp.sync_trigger"))
				Expect(e.Source()).To(Equal(poolARN))
				Expect(e.Subject()).To(Equal(poolID))
				Expect(*user.Username).To(Equal(username))
			})
		})
	})

	When("a client creates a source object with invalid specs", func() {

		// These tests do not require a real pool or sink
		BeforeEach(func() {
			poolARN = "arn:aws:cognito-idp:us-west-2:123456789012:userpool/us-west-2_FaK3p001"

			awsCreds = credentials.Value{
				AccessKeyID:     "fake",
				SecretAccessKey: "fake",
			}

			sink = &duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: "fake/v1",
					Kind:       "Fake",
					Name:       "fake",
				},
			}
		})

		// Here we use
		//   "Specify: the API server rejects ..., By: setting an invalid ..."
		// instead of
		//   "When: it sets an invalid ..., Specify: the API server rejects ..."
		// to avoid creating a namespace for each spec, due to their simplicity.
		Specify("the API server rejects the creation of that object", func() {

			By("setting an invalid ARN", func() {
				invalidARN := "arn:aws:cognito-idp:invalid::"

				_, err := createSource(srcClient, ns, "test-invalid-arn-", sink,
					withARN(invalidARN),
					withCredentials(awsCreds),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.arn: Invalid value: "))
			})

			By("setting empty credentials", func() {
				_, err := createSource(srcClient, ns, "test-nocreds-", sink,
					withARN(poolARN),
					withCredentials(credentials.Value{}),
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					`"spec.auth.credentials.accessKeyID" must validate one and only one schema (oneOf).`))
			})
		})
	})
})

// createSource creates an AWSCognitoUserPoolSource object initialized with the given options.
func createSource(srcClient dynamic.ResourceInterface, namespace, namePrefix string,
	sink *duckv1.Destination, opts ...sourceOption) (*unstructured.Unstructured, error) {

	src := &unstructured.Unstructured{}
	src.SetAPIVersion(sourceAPIVersion.String())
	src.SetKind(sourceKind)
	src.SetNamespace(namespace)
	src.SetGenerateName(namePrefix)

	if err := unstructured.SetNestedMap(src.Object, ducktypes.DestinationToMap(sink), "spec", "sink"); err != nil {
		framework.FailfWithOffset(2, "Failed to set spec.sink field: %s", err)
	}

	for _, opt := range opts {
		opt(src)
	}

	return srcClient.Create(context.Background(), src, metav1.CreateOptions{})
}

type sourceOption func(*unstructured.Unstructured)

func withARN(arn string) sourceOption {
	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedField(src.Object, arn, "spec", "arn"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.arn field: %s", err)
		}
	}
}

func withCredentials(creds credentials.Value) sourceOption {
	credsMap := map[string]interface{}{
		"accessKeyID":     map[string]interface{}{},
		"secretAccessKey": map[string]interface{}{},
	}
	if creds.AccessKeyID != "" {
		credsMap["accessKeyID"] = map[string]interface{}{"value": creds.AccessKeyID}
	}
	if creds.SecretAccessKey != "" {
		credsMap["secretAccessKey"] = map[string]interface{}{"value": creds.SecretAccessKey}
	}

	return func(src *unstructured.Unstructured) {
		if err := unstructured.SetNestedMap(src.Object, credsMap, "spec", "auth", "credentials"); err != nil {
			framework.FailfWithOffset(3, "Failed to set spec.auth.credentials field: %s", err)
		}
	}
}

func readAWSCredentials(sess *session.Session) credentials.Value {
	creds, err := sess.Config.Credentials.Get()
	if err != nil {
		framework.FailfWithOffset(2, "Error reading AWS credentials: %s", err)
	}

	return creds
}

// readReceivedEvents returns a function that reads CloudEvents received by the
// event-display application and stores the result as the value of the given
// `receivedEvents` variable.
// The returned function signature satisfies the contract expected by
// gomega.Eventually: no argument and one or more return values.
func readReceivedEvents(c clientset.Interface, namespace, eventDisplayDeplName string,
	receivedEvents *[]cloudevents.Event) func() []cloudevents.Event {

	return func() []cloudevents.Event {
		ev := bridges.ReceivedEventDisplayEvents(
			apps.GetLogs(c, namespace, eventDisplayDeplName),
		)
		*receivedEvents = ev
		return ev
	}
}
