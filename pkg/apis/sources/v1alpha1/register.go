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
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"

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

// ObjectWebhook is an interface to use in the webhook.
type ObjectWebhook interface {
	runtime.Object
	apis.Validatable
	apis.Defaultable
	kmeta.OwnerRefable
}

// Objects holds instances of ObjectWebhook and runtime.Objects List.
// +k8s:deepcopy-gen=false
type Objects struct {
	Single ObjectWebhook
	List   runtime.Object
}

// AllTypes is a list of all the types defined in this package.
var AllTypes = []Objects{
	{&AWSCloudWatchSource{}, &AWSCloudWatchSourceList{}},
	{&AWSCloudWatchLogsSource{}, &AWSCloudWatchLogsSourceList{}},
	{&AWSCodeCommitSource{}, &AWSCodeCommitSourceList{}},
	{&AWSCognitoIdentitySource{}, &AWSCognitoIdentitySourceList{}},
	{&AWSCognitoUserPoolSource{}, &AWSCognitoUserPoolSourceList{}},
	{&AWSDynamoDBSource{}, &AWSDynamoDBSourceList{}},
	{&AWSEventBridgeSource{}, &AWSEventBridgeSourceList{}},
	{&AWSKinesisSource{}, &AWSKinesisSourceList{}},
	{&AWSS3Source{}, &AWSS3SourceList{}},
	{&AWSSNSSource{}, &AWSSNSSourceList{}},
	{&AWSSQSSource{}, &AWSSQSSourceList{}},
	{&AWSPerformanceInsightsSource{}, &AWSPerformanceInsightsSourceList{}},
	{&AzureActivityLogsSource{}, &AzureActivityLogsSourceList{}},
	{&AzureBlobStorageSource{}, &AzureBlobStorageSourceList{}},
	{&AzureEventGridSource{}, &AzureEventGridSourceList{}},
	{&AzureEventHubsSource{}, &AzureEventHubsSourceList{}},
	{&AzureIOTHubSource{}, &AzureIOTHubSourceList{}},
	{&AzureQueueStorageSource{}, &AzureQueueStorageSourceList{}},
	{&AzureServiceBusQueueSource{}, &AzureServiceBusQueueSourceList{}},
	{&AzureServiceBusTopicSource{}, &AzureServiceBusTopicSourceList{}},
	{&CloudEventsSource{}, &CloudEventsSourceList{}},
	{&KafkaSource{}, &KafkaSourceList{}},
	{&GoogleCloudAuditLogsSource{}, &GoogleCloudAuditLogsSourceList{}},
	{&GoogleCloudBillingSource{}, &GoogleCloudBillingSourceList{}},
	{&GoogleCloudPubSubSource{}, &GoogleCloudPubSubSourceList{}},
	{&GoogleCloudSourceRepositoriesSource{}, &GoogleCloudSourceRepositoriesSourceList{}},
	{&GoogleCloudStorageSource{}, &GoogleCloudStorageSourceList{}},
	{&HTTPPollerSource{}, &HTTPPollerSourceList{}},
	{&IBMMQSource{}, &IBMMQSourceList{}},
	{&OCIMetricsSource{}, &OCIMetricsSourceList{}},
	{&SalesforceSource{}, &SalesforceSourceList{}},
	{&SlackSource{}, &SlackSourceList{}},
	{&SolaceSource{}, &SolaceSourceList{}},
	{&TwilioSource{}, &TwilioSourceList{}},
	{&WebhookSource{}, &WebhookSourceList{}},
	{&ZendeskSource{}, &ZendeskSourceList{}},
}

// addKnownTypes adds all this custom API's types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	allTypes := make([]runtime.Object, 0, len(AllTypes)*2)
	for _, t := range AllTypes {
		allTypes = append(allTypes, t.Single, t.List)
	}
	scheme.AddKnownTypes(SchemeGroupVersion, allTypes...)
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
