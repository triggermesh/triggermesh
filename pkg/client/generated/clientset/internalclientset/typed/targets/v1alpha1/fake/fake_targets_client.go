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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/clientset/internalclientset/typed/targets/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeTargetsV1alpha1 struct {
	*testing.Fake
}

func (c *FakeTargetsV1alpha1) AWSComprehendTargets(namespace string) v1alpha1.AWSComprehendTargetInterface {
	return &FakeAWSComprehendTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) AWSDynamoDBTargets(namespace string) v1alpha1.AWSDynamoDBTargetInterface {
	return &FakeAWSDynamoDBTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) AWSKinesisTargets(namespace string) v1alpha1.AWSKinesisTargetInterface {
	return &FakeAWSKinesisTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) AWSLambdaTargets(namespace string) v1alpha1.AWSLambdaTargetInterface {
	return &FakeAWSLambdaTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) AWSS3Targets(namespace string) v1alpha1.AWSS3TargetInterface {
	return &FakeAWSS3Targets{c, namespace}
}

func (c *FakeTargetsV1alpha1) AWSSNSTargets(namespace string) v1alpha1.AWSSNSTargetInterface {
	return &FakeAWSSNSTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) AWSSQSTargets(namespace string) v1alpha1.AWSSQSTargetInterface {
	return &FakeAWSSQSTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) AlibabaOSSTargets(namespace string) v1alpha1.AlibabaOSSTargetInterface {
	return &FakeAlibabaOSSTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) ConfluentTargets(namespace string) v1alpha1.ConfluentTargetInterface {
	return &FakeConfluentTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) DatadogTargets(namespace string) v1alpha1.DatadogTargetInterface {
	return &FakeDatadogTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) ElasticsearchTargets(namespace string) v1alpha1.ElasticsearchTargetInterface {
	return &FakeElasticsearchTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) GoogleCloudFirestoreTargets(namespace string) v1alpha1.GoogleCloudFirestoreTargetInterface {
	return &FakeGoogleCloudFirestoreTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) GoogleCloudStorageTargets(namespace string) v1alpha1.GoogleCloudStorageTargetInterface {
	return &FakeGoogleCloudStorageTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) GoogleCloudWorkflowsTargets(namespace string) v1alpha1.GoogleCloudWorkflowsTargetInterface {
	return &FakeGoogleCloudWorkflowsTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) GoogleSheetTargets(namespace string) v1alpha1.GoogleSheetTargetInterface {
	return &FakeGoogleSheetTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) HTTPTargets(namespace string) v1alpha1.HTTPTargetInterface {
	return &FakeHTTPTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) HasuraTargets(namespace string) v1alpha1.HasuraTargetInterface {
	return &FakeHasuraTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) InfraTargets(namespace string) v1alpha1.InfraTargetInterface {
	return &FakeInfraTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) JiraTargets(namespace string) v1alpha1.JiraTargetInterface {
	return &FakeJiraTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) LogzMetricsTargets(namespace string) v1alpha1.LogzMetricsTargetInterface {
	return &FakeLogzMetricsTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) LogzTargets(namespace string) v1alpha1.LogzTargetInterface {
	return &FakeLogzTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) OracleTargets(namespace string) v1alpha1.OracleTargetInterface {
	return &FakeOracleTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) SalesforceTargets(namespace string) v1alpha1.SalesforceTargetInterface {
	return &FakeSalesforceTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) SendGridTargets(namespace string) v1alpha1.SendGridTargetInterface {
	return &FakeSendGridTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) SlackTargets(namespace string) v1alpha1.SlackTargetInterface {
	return &FakeSlackTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) SplunkTargets(namespace string) v1alpha1.SplunkTargetInterface {
	return &FakeSplunkTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) TektonTargets(namespace string) v1alpha1.TektonTargetInterface {
	return &FakeTektonTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) TwilioTargets(namespace string) v1alpha1.TwilioTargetInterface {
	return &FakeTwilioTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) UiPathTargets(namespace string) v1alpha1.UiPathTargetInterface {
	return &FakeUiPathTargets{c, namespace}
}

func (c *FakeTargetsV1alpha1) ZendeskTargets(namespace string) v1alpha1.ZendeskTargetInterface {
	return &FakeZendeskTargets{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeTargetsV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
