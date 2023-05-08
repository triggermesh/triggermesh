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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/triggermesh/triggermesh/pkg/apis/common/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/apis/targets"
)

var (
	// SchemeGroupVersion contains the group and version used to register types for this custom API.
	SchemeGroupVersion = schema.GroupVersion{Group: targets.GroupName, Version: "v1alpha1"}
	// SchemeBuilder creates a Scheme builder that is used to register types for this custom API.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme registers the types stored in SchemeBuilder.
	AddToScheme = SchemeBuilder.AddToScheme
)

// AllTypes is a list of all the types defined in this package.
var AllTypes = []v1alpha1.GroupObject{
	{Single: &AWSComprehendTarget{}, List: &AWSComprehendTargetList{}},
	{Single: &AWSDynamoDBTarget{}, List: &AWSDynamoDBTargetList{}},
	{Single: &AWSEventBridgeTarget{}, List: &AWSEventBridgeTargetList{}},
	{Single: &AWSKinesisTarget{}, List: &AWSKinesisTargetList{}},
	{Single: &AWSLambdaTarget{}, List: &AWSLambdaTargetList{}},
	{Single: &AWSS3Target{}, List: &AWSS3TargetList{}},
	{Single: &AWSSNSTarget{}, List: &AWSSNSTargetList{}},
	{Single: &AWSSQSTarget{}, List: &AWSSQSTargetList{}},
	{Single: &AzureEventHubsTarget{}, List: &AzureEventHubsTargetList{}},
	{Single: &AzureSentinelTarget{}, List: &AzureSentinelTargetList{}},
	{Single: &AzureServiceBusTarget{}, List: &AzureServiceBusTargetList{}},
	{Single: &CloudEventsTarget{}, List: &CloudEventsTargetList{}},
	{Single: &DatadogTarget{}, List: &DatadogTargetList{}},
	{Single: &ElasticsearchTarget{}, List: &ElasticsearchTargetList{}},
	{Single: &GoogleCloudFirestoreTarget{}, List: &GoogleCloudFirestoreTargetList{}},
	{Single: &GoogleCloudStorageTarget{}, List: &GoogleCloudStorageTargetList{}},
	{Single: &GoogleCloudWorkflowsTarget{}, List: &GoogleCloudWorkflowsTargetList{}},
	{Single: &GoogleCloudPubSubTarget{}, List: &GoogleCloudPubSubTargetList{}},
	{Single: &GoogleSheetTarget{}, List: &GoogleSheetTargetList{}},
	{Single: &HTTPTarget{}, List: &HTTPTargetList{}},
	{Single: &IBMMQTarget{}, List: &IBMMQTargetList{}},
	{Single: &JiraTarget{}, List: &JiraTargetList{}},
	{Single: &KafkaTarget{}, List: &KafkaTargetList{}},
	{Single: &LogzMetricsTarget{}, List: &LogzMetricsTargetList{}},
	{Single: &LogzTarget{}, List: &LogzTargetList{}},
	{Single: &MongoDBTarget{}, List: &MongoDBTargetList{}},
	{Single: &OracleTarget{}, List: &OracleTargetList{}},
	{Single: &SalesforceTarget{}, List: &SalesforceTargetList{}},
	{Single: &SendGridTarget{}, List: &SendGridTargetList{}},
	{Single: &SlackTarget{}, List: &SlackTargetList{}},
	{Single: &SolaceTarget{}, List: &SolaceTargetList{}},
	{Single: &SplunkTarget{}, List: &SplunkTargetList{}},
	{Single: &TwilioTarget{}, List: &TwilioTargetList{}},
	{Single: &ZendeskTarget{}, List: &ZendeskTargetList{}},
}

// addKnownTypes adds all this custom API's types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	for _, t := range AllTypes {
		scheme.AddKnownTypes(SchemeGroupVersion, t.Single, t.List)
	}
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind.
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
