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
	"github.com/triggermesh/triggermesh/pkg/apis/sources"
)

var (
	// SchemeGroupVersion contains the group and version used to register types for this custom API.
	SchemeGroupVersion = schema.GroupVersion{Group: sources.GroupName, Version: "v1alpha1"}
	// SchemeBuilder creates a Scheme builder that is used to register types for this custom API.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme registers the types stored in SchemeBuilder.
	AddToScheme = SchemeBuilder.AddToScheme
)

// AllTypes is a list of all the types defined in this package.
var AllTypes = []v1alpha1.GroupObject{
	{Single: &AWSCloudWatchSource{}, List: &AWSCloudWatchSourceList{}},
	{Single: &AWSCloudWatchLogsSource{}, List: &AWSCloudWatchLogsSourceList{}},
	{Single: &AWSCodeCommitSource{}, List: &AWSCodeCommitSourceList{}},
	{Single: &AWSCognitoIdentitySource{}, List: &AWSCognitoIdentitySourceList{}},
	{Single: &AWSCognitoUserPoolSource{}, List: &AWSCognitoUserPoolSourceList{}},
	{Single: &AWSDynamoDBSource{}, List: &AWSDynamoDBSourceList{}},
	{Single: &AWSEventBridgeSource{}, List: &AWSEventBridgeSourceList{}},
	{Single: &AWSKinesisSource{}, List: &AWSKinesisSourceList{}},
	{Single: &AWSS3Source{}, List: &AWSS3SourceList{}},
	{Single: &AWSSNSSource{}, List: &AWSSNSSourceList{}},
	{Single: &AWSSQSSource{}, List: &AWSSQSSourceList{}},
	{Single: &AWSPerformanceInsightsSource{}, List: &AWSPerformanceInsightsSourceList{}},
	{Single: &AzureActivityLogsSource{}, List: &AzureActivityLogsSourceList{}},
	{Single: &AzureBlobStorageSource{}, List: &AzureBlobStorageSourceList{}},
	{Single: &AzureEventGridSource{}, List: &AzureEventGridSourceList{}},
	{Single: &AzureEventHubsSource{}, List: &AzureEventHubsSourceList{}},
	{Single: &AzureIOTHubSource{}, List: &AzureIOTHubSourceList{}},
	{Single: &AzureQueueStorageSource{}, List: &AzureQueueStorageSourceList{}},
	{Single: &AzureServiceBusQueueSource{}, List: &AzureServiceBusQueueSourceList{}},
	{Single: &AzureServiceBusSource{}, List: &AzureServiceBusSourceList{}},
	{Single: &AzureServiceBusTopicSource{}, List: &AzureServiceBusTopicSourceList{}},
	{Single: &CloudEventsSource{}, List: &CloudEventsSourceList{}},
	{Single: &MongoDBSource{}, List: &MongoDBSourceList{}},
	{Single: &KafkaSource{}, List: &KafkaSourceList{}},
	{Single: &GoogleCloudAuditLogsSource{}, List: &GoogleCloudAuditLogsSourceList{}},
	{Single: &GoogleCloudBillingSource{}, List: &GoogleCloudBillingSourceList{}},
	{Single: &GoogleCloudPubSubSource{}, List: &GoogleCloudPubSubSourceList{}},
	{Single: &GoogleCloudSourceRepositoriesSource{}, List: &GoogleCloudSourceRepositoriesSourceList{}},
	{Single: &GoogleCloudStorageSource{}, List: &GoogleCloudStorageSourceList{}},
	{Single: &HTTPPollerSource{}, List: &HTTPPollerSourceList{}},
	{Single: &IBMMQSource{}, List: &IBMMQSourceList{}},
	{Single: &OCIMetricsSource{}, List: &OCIMetricsSourceList{}},
	{Single: &SalesforceSource{}, List: &SalesforceSourceList{}},
	{Single: &SlackSource{}, List: &SlackSourceList{}},
	{Single: &SolaceSource{}, List: &SolaceSourceList{}},
	{Single: &TwilioSource{}, List: &TwilioSourceList{}},
	{Single: &WebhookSource{}, List: &WebhookSourceList{}},
	{Single: &ZendeskSource{}, List: &ZendeskSourceList{}},
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
