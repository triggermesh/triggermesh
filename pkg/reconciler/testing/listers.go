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

package testing

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	appslistersv1 "k8s.io/client-go/listers/apps/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	rbaclistersv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"

	fakeeventingclientset "knative.dev/eventing/pkg/client/clientset/versioned/fake"
	rt "knative.dev/pkg/reconciler/testing"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	fakeservingclient "knative.dev/serving/pkg/client/clientset/versioned/fake"
	servinglistersv1 "knative.dev/serving/pkg/client/listers/serving/v1"

	extensionsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/extensions/v1alpha1"
	flowv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	sourcesv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	targetsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	fakeclient "github.com/triggermesh/triggermesh/pkg/client/generated/clientset/internalclientset/fake"
	extensionslistersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/extensions/v1alpha1"
	flowlistersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/flow/v1alpha1"
	sourceslistersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/sources/v1alpha1"
	targetslistersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/targets/v1alpha1"
)

var clientSetSchemes = []func(*runtime.Scheme) error{
	fakeclient.AddToScheme,
	fakek8sclient.AddToScheme,
	fakeservingclient.AddToScheme,
	// although our reconcilers do not handle eventing objects directly, we
	// do need to register the eventing Scheme so that sink URI resolvers
	// can recognize the Broker objects we use in tests
	fakeeventingclientset.AddToScheme,
}

// NewScheme returns a new scheme populated with the types defined in clientSetSchemes.
func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	sb := runtime.NewSchemeBuilder(clientSetSchemes...)
	if err := sb.AddToScheme(scheme); err != nil {
		panic(fmt.Errorf("error building Scheme: %s", err))
	}

	return scheme
}

// Listers returns listers and objects filtered from those listers.
type Listers struct {
	sorter rt.ObjectSorter
}

// NewListers returns a new instance of Listers initialized with the given objects.
func NewListers(scheme *runtime.Scheme, objs []runtime.Object) Listers {
	ls := Listers{
		sorter: rt.NewObjectSorter(scheme),
	}

	ls.sorter.AddObjects(objs...)

	return ls
}

// IndexerFor returns the indexer for the given object.
func (l *Listers) IndexerFor(obj runtime.Object) cache.Indexer {
	return l.sorter.IndexerForObjectType(obj)
}

// GetTriggerMeshObjects returns objects from TriggerMesh APIs.
func (l *Listers) GetTriggerMeshObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakeclient.AddToScheme)
}

// GetKubeObjects returns objects from Kubernetes APIs.
func (l *Listers) GetKubeObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakek8sclient.AddToScheme)
}

// GetServingObjects returns objects from the serving API.
func (l *Listers) GetServingObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakeservingclient.AddToScheme)
}

// GetDeploymentLister returns a lister for Deployment objects.
func (l *Listers) GetDeploymentLister() appslistersv1.DeploymentLister {
	return appslistersv1.NewDeploymentLister(l.IndexerFor(&appsv1.Deployment{}))
}

// GetPodLister returns a lister for Pod objects.
func (l *Listers) GetPodLister() corelistersv1.PodLister {
	return corelistersv1.NewPodLister(l.IndexerFor(&corev1.Pod{}))
}

// GetServiceLister returns a lister for Service objects.
func (l *Listers) GetServiceLister() servinglistersv1.ServiceLister {
	return servinglistersv1.NewServiceLister(l.IndexerFor(&servingv1.Service{}))
}

// GetServiceAccountLister returns a lister for ServiceAccount objects.
func (l *Listers) GetServiceAccountLister() corelistersv1.ServiceAccountLister {
	return corelistersv1.NewServiceAccountLister(l.IndexerFor(&corev1.ServiceAccount{}))
}

// GetRoleBindingLister returns a lister for RoleBinding objects
func (l *Listers) GetRoleBindingLister() rbaclistersv1.RoleBindingLister {
	return rbaclistersv1.NewRoleBindingLister(l.IndexerFor(&rbacv1.RoleBinding{}))
}

// GetJQTransformationLister returns a Lister for JQTransformation objects.
func (l *Listers) GetJQTransformationLister() flowlistersv1alpha1.JQTransformationLister {
	return flowlistersv1alpha1.NewJQTransformationLister(l.IndexerFor(&flowv1alpha1.JQTransformation{}))
}

// GetSynchronizerLister returns a Lister for Synchronizer objects.
func (l *Listers) GetSynchronizerLister() flowlistersv1alpha1.SynchronizerLister {
	return flowlistersv1alpha1.NewSynchronizerLister(l.IndexerFor(&flowv1alpha1.Synchronizer{}))
}

// GetTransformationLister returns a Lister for Transformation objects.
func (l *Listers) GetTransformationLister() flowlistersv1alpha1.TransformationLister {
	return flowlistersv1alpha1.NewTransformationLister(l.IndexerFor(&flowv1alpha1.Transformation{}))
}

// GetXMLToJSONTransformationLister returns a Lister for XMLToJSONTransformation objects.
func (l *Listers) GetXMLToJSONTransformationLister() flowlistersv1alpha1.XMLToJSONTransformationLister {
	return flowlistersv1alpha1.NewXMLToJSONTransformationLister(l.IndexerFor(&flowv1alpha1.XMLToJSONTransformation{}))
}

// GetXSLTTransformationLister returns a Lister for XSLTTransformation objects.
func (l *Listers) GetXSLTTransformationLister() flowlistersv1alpha1.XSLTTransformationLister {
	return flowlistersv1alpha1.NewXSLTTransformationLister(l.IndexerFor(&flowv1alpha1.XSLTTransformation{}))
}

// GetAWSCloudWatchSourceLister returns a Lister for AWSCloudWatchSource objects.
func (l *Listers) GetAWSCloudWatchSourceLister() sourceslistersv1alpha1.AWSCloudWatchSourceLister {
	return sourceslistersv1alpha1.NewAWSCloudWatchSourceLister(l.IndexerFor(&sourcesv1alpha1.AWSCloudWatchSource{}))
}

// GetAWSCloudWatchLogsSourceLister returns a Lister for AWSCloudWatchLogsSource objects.
func (l *Listers) GetAWSCloudWatchLogsSourceLister() sourceslistersv1alpha1.AWSCloudWatchLogsSourceLister {
	return sourceslistersv1alpha1.NewAWSCloudWatchLogsSourceLister(l.IndexerFor(&sourcesv1alpha1.AWSCloudWatchLogsSource{}))
}

// GetAWSCodeCommitSourceLister returns a Lister for AWSCodeCommitSource objects.
func (l *Listers) GetAWSCodeCommitSourceLister() sourceslistersv1alpha1.AWSCodeCommitSourceLister {
	return sourceslistersv1alpha1.NewAWSCodeCommitSourceLister(l.IndexerFor(&sourcesv1alpha1.AWSCodeCommitSource{}))
}

// GetAWSCognitoUserPoolSourceLister returns a Lister for AWSCognitoIdentitySource objects.
func (l *Listers) GetAWSCognitoUserPoolSourceLister() sourceslistersv1alpha1.AWSCognitoUserPoolSourceLister {
	return sourceslistersv1alpha1.NewAWSCognitoUserPoolSourceLister(l.IndexerFor(&sourcesv1alpha1.AWSCognitoUserPoolSource{}))
}

// GetAWSCognitoIdentitySourceLister returns a Lister for AWSCognitoUserPoolSource objects.
func (l *Listers) GetAWSCognitoIdentitySourceLister() sourceslistersv1alpha1.AWSCognitoIdentitySourceLister {
	return sourceslistersv1alpha1.NewAWSCognitoIdentitySourceLister(l.IndexerFor(&sourcesv1alpha1.AWSCognitoIdentitySource{}))
}

// GetAWSDynamoDBSourceLister returns a Lister for AWSDynamoDBSource objects.
func (l *Listers) GetAWSDynamoDBSourceLister() sourceslistersv1alpha1.AWSDynamoDBSourceLister {
	return sourceslistersv1alpha1.NewAWSDynamoDBSourceLister(l.IndexerFor(&sourcesv1alpha1.AWSDynamoDBSource{}))
}

// GetAWSEventBridgeSourceLister returns a Lister for AWSEventBridgeSource objects.
func (l *Listers) GetAWSEventBridgeSourceLister() sourceslistersv1alpha1.AWSEventBridgeSourceLister {
	return sourceslistersv1alpha1.NewAWSEventBridgeSourceLister(l.IndexerFor(&sourcesv1alpha1.AWSEventBridgeSource{}))
}

// GetAWSKinesisSourceLister returns a Lister for AWSKinesisSource objects.
func (l *Listers) GetAWSKinesisSourceLister() sourceslistersv1alpha1.AWSKinesisSourceLister {
	return sourceslistersv1alpha1.NewAWSKinesisSourceLister(l.IndexerFor(&sourcesv1alpha1.AWSKinesisSource{}))
}

// GetAWSSNSSourceLister returns a Lister for AWSSNSSource objects.
func (l *Listers) GetAWSSNSSourceLister() sourceslistersv1alpha1.AWSSNSSourceLister {
	return sourceslistersv1alpha1.NewAWSSNSSourceLister(l.IndexerFor(&sourcesv1alpha1.AWSSNSSource{}))
}

// GetAWSSQSSourceLister returns a Lister for AWSSQSSource objects.
func (l *Listers) GetAWSSQSSourceLister() sourceslistersv1alpha1.AWSSQSSourceLister {
	return sourceslistersv1alpha1.NewAWSSQSSourceLister(l.IndexerFor(&sourcesv1alpha1.AWSSQSSource{}))
}

// GetAzureActivityLogsSourceLister returns a Lister for AzureActivityLogsSource objects.
func (l *Listers) GetAzureActivityLogsSourceLister() sourceslistersv1alpha1.AzureActivityLogsSourceLister {
	return sourceslistersv1alpha1.NewAzureActivityLogsSourceLister(l.IndexerFor(&sourcesv1alpha1.AzureActivityLogsSource{}))
}

// GetAzureBlobStorageSourceLister returns a Lister for AzureBlobStorageSource objects.
func (l *Listers) GetAzureBlobStorageSourceLister() sourceslistersv1alpha1.AzureBlobStorageSourceLister {
	return sourceslistersv1alpha1.NewAzureBlobStorageSourceLister(l.IndexerFor(&sourcesv1alpha1.AzureBlobStorageSource{}))
}

// GetAzureEventGridSourceLister returns a Lister for AzureEventGridSource objects.
func (l *Listers) GetAzureEventGridSourceLister() sourceslistersv1alpha1.AzureEventGridSourceLister {
	return sourceslistersv1alpha1.NewAzureEventGridSourceLister(l.IndexerFor(&sourcesv1alpha1.AzureEventGridSource{}))
}

// GetAzureEventHubsSourceLister returns a Lister for AzureEventHubsSource objects.
func (l *Listers) GetAzureEventHubsSourceLister() sourceslistersv1alpha1.AzureEventHubsSourceLister {
	return sourceslistersv1alpha1.NewAzureEventHubsSourceLister(l.IndexerFor(&sourcesv1alpha1.AzureEventHubsSource{}))
}

// GetAzureIOTHubSourceLister returns a Lister for AzureIOTHuSource objects.
func (l *Listers) GetAzureIOTHubSourceLister() sourceslistersv1alpha1.AzureIOTHubSourceLister {
	return sourceslistersv1alpha1.NewAzureIOTHubSourceLister(l.IndexerFor(&sourcesv1alpha1.AzureIOTHubSource{}))
}

// GetAzureServiceBusQueueSourceLister returns a Lister for AzureServiceBusQueueSource objects.
func (l *Listers) GetAzureServiceBusQueueSourceLister() sourceslistersv1alpha1.AzureServiceBusQueueSourceLister {
	return sourceslistersv1alpha1.NewAzureServiceBusQueueSourceLister(l.IndexerFor(&sourcesv1alpha1.AzureServiceBusQueueSource{}))
}

// GetAzureServiceBusSourceLister returns a Lister for AzureServiceBusSource objects.
func (l *Listers) GetAzureServiceBusSourceLister() sourceslistersv1alpha1.AzureServiceBusSourceLister {
	return sourceslistersv1alpha1.NewAzureServiceBusSourceLister(l.IndexerFor(&sourcesv1alpha1.AzureServiceBusSource{}))
}

// GetAzureServiceBusTopicSourceLister returns a Lister for AzureServiceBusTopicSource objects.
func (l *Listers) GetAzureServiceBusTopicSourceLister() sourceslistersv1alpha1.AzureServiceBusTopicSourceLister {
	return sourceslistersv1alpha1.NewAzureServiceBusTopicSourceLister(l.IndexerFor(&sourcesv1alpha1.AzureServiceBusTopicSource{}))
}

// GetCloudEventsSourceLister returns a Lister for CloudEventsSource objects.
func (l *Listers) GetCloudEventsSourceLister() sourceslistersv1alpha1.CloudEventsSourceLister {
	return sourceslistersv1alpha1.NewCloudEventsSourceLister(l.IndexerFor(&sourcesv1alpha1.CloudEventsSource{}))
}

// GetGoogleCloudAuditLogsSourceLister returns a Lister for GoogleCloudAuditLogsSource objects.
func (l *Listers) GetGoogleCloudAuditLogsSourceLister() sourceslistersv1alpha1.GoogleCloudAuditLogsSourceLister {
	return sourceslistersv1alpha1.NewGoogleCloudAuditLogsSourceLister(l.IndexerFor(&sourcesv1alpha1.GoogleCloudAuditLogsSource{}))
}

// GetGoogleCloudBillingSourceLister returns a Lister for GoogleCloudBillingSource objects.
func (l *Listers) GetGoogleCloudBillingSourceLister() sourceslistersv1alpha1.GoogleCloudBillingSourceLister {
	return sourceslistersv1alpha1.NewGoogleCloudBillingSourceLister(l.IndexerFor(&sourcesv1alpha1.GoogleCloudBillingSource{}))
}

// GetGoogleCloudPubSubSourceLister returns a Lister for GoogleCloudPubSubSource objects.
func (l *Listers) GetGoogleCloudPubSubSourceLister() sourceslistersv1alpha1.GoogleCloudPubSubSourceLister {
	return sourceslistersv1alpha1.NewGoogleCloudPubSubSourceLister(l.IndexerFor(&sourcesv1alpha1.GoogleCloudPubSubSource{}))
}

// GetGoogleCloudSourceRepositoriesSourceLister returns a Lister for GoogleCloudSourceRepositoriesSource objects.
func (l *Listers) GetGoogleCloudSourceRepositoriesSourceLister() sourceslistersv1alpha1.GoogleCloudSourceRepositoriesSourceLister {
	return sourceslistersv1alpha1.NewGoogleCloudSourceRepositoriesSourceLister(l.IndexerFor(&sourcesv1alpha1.GoogleCloudSourceRepositoriesSource{}))
}

// GetGoogleCloudStorageSourceLister returns a Lister for GoogleCloudStorageSource objects.
func (l *Listers) GetGoogleCloudStorageSourceLister() sourceslistersv1alpha1.GoogleCloudStorageSourceLister {
	return sourceslistersv1alpha1.NewGoogleCloudStorageSourceLister(l.IndexerFor(&sourcesv1alpha1.GoogleCloudStorageSource{}))
}

// GetHTTPPollerSourceLister returns a Lister for HTTPPollerSource objects.
func (l *Listers) GetHTTPPollerSourceLister() sourceslistersv1alpha1.HTTPPollerSourceLister {
	return sourceslistersv1alpha1.NewHTTPPollerSourceLister(l.IndexerFor(&sourcesv1alpha1.HTTPPollerSource{}))
}

// GetKafkaSourceLister returns a Lister for KafkaSource objects.
func (l *Listers) GetKafkaSourceLister() sourceslistersv1alpha1.KafkaSourceLister {
	return sourceslistersv1alpha1.NewKafkaSourceLister(l.IndexerFor(&sourcesv1alpha1.KafkaSource{}))
}

// GetOCIMetricsSourceLister returns a Lister for OCIMetricsSource objects.
func (l *Listers) GetOCIMetricsSourceLister() sourceslistersv1alpha1.OCIMetricsSourceLister {
	return sourceslistersv1alpha1.NewOCIMetricsSourceLister(l.IndexerFor(&sourcesv1alpha1.OCIMetricsSource{}))
}

// GetSalesforceSourceLister returns a Lister for SalesforceSource objects.
func (l *Listers) GetSalesforceSourceLister() sourceslistersv1alpha1.SalesforceSourceLister {
	return sourceslistersv1alpha1.NewSalesforceSourceLister(l.IndexerFor(&sourcesv1alpha1.SalesforceSource{}))
}

// GetSlackSourceLister returns a Lister for SlackSource objects.
func (l *Listers) GetSlackSourceLister() sourceslistersv1alpha1.SlackSourceLister {
	return sourceslistersv1alpha1.NewSlackSourceLister(l.IndexerFor(&sourcesv1alpha1.SlackSource{}))
}

// GetSolaceSourceLister returns a Lister for SolaceSource objects.
func (l *Listers) GetSolaceSourceLister() sourceslistersv1alpha1.SolaceSourceLister {
	return sourceslistersv1alpha1.NewSolaceSourceLister(l.IndexerFor(&sourcesv1alpha1.SolaceSource{}))
}

// GetTwilioSourceLister returns a Lister for TwilioSource objects.
func (l *Listers) GetTwilioSourceLister() sourceslistersv1alpha1.TwilioSourceLister {
	return sourceslistersv1alpha1.NewTwilioSourceLister(l.IndexerFor(&sourcesv1alpha1.TwilioSource{}))
}

// GetWebhookSourceLister returns a Lister for WebhookSource objects.
func (l *Listers) GetWebhookSourceLister() sourceslistersv1alpha1.WebhookSourceLister {
	return sourceslistersv1alpha1.NewWebhookSourceLister(l.IndexerFor(&sourcesv1alpha1.WebhookSource{}))
}

// GetZendeskSourceLister returns a Lister for ZendeskSource objects.
func (l *Listers) GetZendeskSourceLister() sourceslistersv1alpha1.ZendeskSourceLister {
	return sourceslistersv1alpha1.NewZendeskSourceLister(l.IndexerFor(&sourcesv1alpha1.ZendeskSource{}))
}

// GetAWSComprehendTargetLister returns a Lister for AWSComprehendTarget objects.
func (l *Listers) GetAWSComprehendTargetLister() targetslistersv1alpha1.AWSComprehendTargetLister {
	return targetslistersv1alpha1.NewAWSComprehendTargetLister(l.IndexerFor(&targetsv1alpha1.AWSComprehendTarget{}))
}

// GetAWSDynamoDBTargetLister returns a Lister for AWSDynamoDBTarget objects.
func (l *Listers) GetAWSDynamoDBTargetLister() targetslistersv1alpha1.AWSDynamoDBTargetLister {
	return targetslistersv1alpha1.NewAWSDynamoDBTargetLister(l.IndexerFor(&targetsv1alpha1.AWSDynamoDBTarget{}))
}

// GetAWSEventBridgeTargetLister returns a Lister for AWSEventBridgeTarget objects.
func (l *Listers) GetAWSEventBridgeTargetLister() targetslistersv1alpha1.AWSEventBridgeTargetLister {
	return targetslistersv1alpha1.NewAWSEventBridgeTargetLister(l.IndexerFor(&targetsv1alpha1.AWSEventBridgeTarget{}))
}

// GetAWSKinesisTargetLister returns a Lister for AWSKinesisTarget objects.
func (l *Listers) GetAWSKinesisTargetLister() targetslistersv1alpha1.AWSKinesisTargetLister {
	return targetslistersv1alpha1.NewAWSKinesisTargetLister(l.IndexerFor(&targetsv1alpha1.AWSKinesisTarget{}))
}

// GetAWSLambdaTargetLister returns a Lister for AWSLambdaTarget objects.
func (l *Listers) GetAWSLambdaTargetLister() targetslistersv1alpha1.AWSLambdaTargetLister {
	return targetslistersv1alpha1.NewAWSLambdaTargetLister(l.IndexerFor(&targetsv1alpha1.AWSLambdaTarget{}))
}

// GetAWSS3TargetLister returns a Lister for AWSS3Target objects.
func (l *Listers) GetAWSS3TargetLister() targetslistersv1alpha1.AWSS3TargetLister {
	return targetslistersv1alpha1.NewAWSS3TargetLister(l.IndexerFor(&targetsv1alpha1.AWSS3Target{}))
}

// GetAWSSNSTargetLister returns a Lister for AWSSNSTarget objects.
func (l *Listers) GetAWSSNSTargetLister() targetslistersv1alpha1.AWSSNSTargetLister {
	return targetslistersv1alpha1.NewAWSSNSTargetLister(l.IndexerFor(&targetsv1alpha1.AWSSNSTarget{}))
}

// GetAWSSQSTargetLister returns a Lister for AWSSQSTarget objects.
func (l *Listers) GetAWSSQSTargetLister() targetslistersv1alpha1.AWSSQSTargetLister {
	return targetslistersv1alpha1.NewAWSSQSTargetLister(l.IndexerFor(&targetsv1alpha1.AWSSQSTarget{}))
}

// GetAzureEventHubsTargetLister returns a Lister for AzureEventHubsTarget objects.
func (l *Listers) GetAzureEventHubsTargetLister() targetslistersv1alpha1.AzureEventHubsTargetLister {
	return targetslistersv1alpha1.NewAzureEventHubsTargetLister(l.IndexerFor(&targetsv1alpha1.AzureEventHubsTarget{}))
}

// GetAzureSentinelTargetLister returns a Lister for AzureSentinelTarget objects.
func (l *Listers) GetAzureSentinelTargetLister() targetslistersv1alpha1.AzureSentinelTargetLister {
	return targetslistersv1alpha1.NewAzureSentinelTargetLister(l.IndexerFor(&targetsv1alpha1.AzureSentinelTarget{}))
}

// GetAzureServiceBusTargetLister returns a Lister for AzureServiceBusTarget objects.
func (l *Listers) GetAzureServiceBusTargetLister() targetslistersv1alpha1.AzureServiceBusTargetLister {
	return targetslistersv1alpha1.NewAzureServiceBusTargetLister(l.IndexerFor(&targetsv1alpha1.AzureServiceBusTarget{}))
}

// GetCloudEventsTargetLister returns a Lister for CloudEventsTarget objects.
func (l *Listers) GetCloudEventsTargetLister() targetslistersv1alpha1.CloudEventsTargetLister {
	return targetslistersv1alpha1.NewCloudEventsTargetLister(l.IndexerFor(&targetsv1alpha1.CloudEventsTarget{}))
}

// GetDatadogTargetLister returns a Lister for DatadogTarget objects.
func (l *Listers) GetDatadogTargetLister() targetslistersv1alpha1.DatadogTargetLister {
	return targetslistersv1alpha1.NewDatadogTargetLister(l.IndexerFor(&targetsv1alpha1.DatadogTarget{}))
}

// GetElasticsearchTargetLister returns a Lister for ElasticsearchTarget objects.
func (l *Listers) GetElasticsearchTargetLister() targetslistersv1alpha1.ElasticsearchTargetLister {
	return targetslistersv1alpha1.NewElasticsearchTargetLister(l.IndexerFor(&targetsv1alpha1.ElasticsearchTarget{}))
}

// GetGoogleCloudFirestoreTargetLister returns a Lister for GoogleCloudFirestoreTarget objects.
func (l *Listers) GetGoogleCloudFirestoreTargetLister() targetslistersv1alpha1.GoogleCloudFirestoreTargetLister {
	return targetslistersv1alpha1.NewGoogleCloudFirestoreTargetLister(l.IndexerFor(&targetsv1alpha1.GoogleCloudFirestoreTarget{}))
}

// GetGoogleCloudPubSubTargetLister returns a Lister for GoogleCloudPubSubTarget objects.
func (l *Listers) GetGoogleCloudPubSubTargetLister() targetslistersv1alpha1.GoogleCloudPubSubTargetLister {
	return targetslistersv1alpha1.NewGoogleCloudPubSubTargetLister(l.IndexerFor(&targetsv1alpha1.GoogleCloudPubSubTarget{}))
}

// GetGoogleCloudStorageTargetLister returns a Lister for GoogleCloudStorageTarget objects.
func (l *Listers) GetGoogleCloudStorageTargetLister() targetslistersv1alpha1.GoogleCloudStorageTargetLister {
	return targetslistersv1alpha1.NewGoogleCloudStorageTargetLister(l.IndexerFor(&targetsv1alpha1.GoogleCloudStorageTarget{}))
}

// GetGoogleCloudWorkflowsTargetLister returns a Lister for GoogleCloudWorkflowsTarget objects.
func (l *Listers) GetGoogleCloudWorkflowsTargetLister() targetslistersv1alpha1.GoogleCloudWorkflowsTargetLister {
	return targetslistersv1alpha1.NewGoogleCloudWorkflowsTargetLister(l.IndexerFor(&targetsv1alpha1.GoogleCloudWorkflowsTarget{}))
}

// GetGoogleSheetTargetLister returns a Lister for GoogleSheetTarget objects.
func (l *Listers) GetGoogleSheetTargetLister() targetslistersv1alpha1.GoogleSheetTargetLister {
	return targetslistersv1alpha1.NewGoogleSheetTargetLister(l.IndexerFor(&targetsv1alpha1.GoogleSheetTarget{}))
}

// GetHTTPTargetLister returns a Lister for HTTPTarget objects.
func (l *Listers) GetHTTPTargetLister() targetslistersv1alpha1.HTTPTargetLister {
	return targetslistersv1alpha1.NewHTTPTargetLister(l.IndexerFor(&targetsv1alpha1.HTTPTarget{}))
}

// GetIBMMQTargetLister returns a Lister for IBMMQTarget objects.
func (l *Listers) GetIBMMQTargetLister() targetslistersv1alpha1.IBMMQTargetLister {
	return targetslistersv1alpha1.NewIBMMQTargetLister(l.IndexerFor(&targetsv1alpha1.IBMMQTarget{}))
}

// GetJiraTargetLister returns a Lister for JiraTarget objects.
func (l *Listers) GetJiraTargetLister() targetslistersv1alpha1.JiraTargetLister {
	return targetslistersv1alpha1.NewJiraTargetLister(l.IndexerFor(&targetsv1alpha1.JiraTarget{}))
}

// GetKafkaTargetLister returns a Lister for KafkaTarget objects.
func (l *Listers) GetKafkaTargetLister() targetslistersv1alpha1.KafkaTargetLister {
	return targetslistersv1alpha1.NewKafkaTargetLister(l.IndexerFor(&targetsv1alpha1.KafkaTarget{}))
}

// GetLogzMetricsTargetLister returns a Lister for LogzMetricsTarget objects.
func (l *Listers) GetLogzMetricsTargetLister() targetslistersv1alpha1.LogzMetricsTargetLister {
	return targetslistersv1alpha1.NewLogzMetricsTargetLister(l.IndexerFor(&targetsv1alpha1.LogzMetricsTarget{}))
}

// GetLogzTargetLister returns a Lister for LogzTarget objects.
func (l *Listers) GetLogzTargetLister() targetslistersv1alpha1.LogzTargetLister {
	return targetslistersv1alpha1.NewLogzTargetLister(l.IndexerFor(&targetsv1alpha1.LogzTarget{}))
}

// GetMongoDBTargetLister returns a Lister for MongoDBTarget objects.
func (l *Listers) GetMongoDBTargetLister() targetslistersv1alpha1.MongoDBTargetLister {
	return targetslistersv1alpha1.NewMongoDBTargetLister(l.IndexerFor(&targetsv1alpha1.MongoDBTarget{}))
}

// GetOracleTargetLister returns a Lister for OracleTarget objects.
func (l *Listers) GetOracleTargetLister() targetslistersv1alpha1.OracleTargetLister {
	return targetslistersv1alpha1.NewOracleTargetLister(l.IndexerFor(&targetsv1alpha1.OracleTarget{}))
}

// GetSalesforceTargetLister returns a Lister for SalesforceTarget objects.
func (l *Listers) GetSalesforceTargetLister() targetslistersv1alpha1.SalesforceTargetLister {
	return targetslistersv1alpha1.NewSalesforceTargetLister(l.IndexerFor(&targetsv1alpha1.SalesforceTarget{}))
}

// GetSendGridTargetLister returns a Lister for SendGridTarget objects.
func (l *Listers) GetSendGridTargetLister() targetslistersv1alpha1.SendGridTargetLister {
	return targetslistersv1alpha1.NewSendGridTargetLister(l.IndexerFor(&targetsv1alpha1.SendGridTarget{}))
}

// GetSlackTargetLister returns a Lister for SlackTarget objects.
func (l *Listers) GetSlackTargetLister() targetslistersv1alpha1.SlackTargetLister {
	return targetslistersv1alpha1.NewSlackTargetLister(l.IndexerFor(&targetsv1alpha1.SlackTarget{}))
}

// GetSolaceTargetLister returns a Lister for SolaceTarget objects.
func (l *Listers) GetSolaceTargetLister() targetslistersv1alpha1.SolaceTargetLister {
	return targetslistersv1alpha1.NewSolaceTargetLister(l.IndexerFor(&targetsv1alpha1.SolaceTarget{}))
}

// GetSplunkTargetLister returns a Lister for SplunkTarget objects.
func (l *Listers) GetSplunkTargetLister() targetslistersv1alpha1.SplunkTargetLister {
	return targetslistersv1alpha1.NewSplunkTargetLister(l.IndexerFor(&targetsv1alpha1.SplunkTarget{}))
}

// GetTwilioTargetLister returns a Lister for TwilioTarget objects.
func (l *Listers) GetTwilioTargetLister() targetslistersv1alpha1.TwilioTargetLister {
	return targetslistersv1alpha1.NewTwilioTargetLister(l.IndexerFor(&targetsv1alpha1.TwilioTarget{}))
}

// GetZendeskTargetLister returns a Lister for ZendeskTarget objects.
func (l *Listers) GetZendeskTargetLister() targetslistersv1alpha1.ZendeskTargetLister {
	return targetslistersv1alpha1.NewZendeskTargetLister(l.IndexerFor(&targetsv1alpha1.ZendeskTarget{}))
}

// GetFunctionLister returns a Lister for Function objects.
func (l *Listers) GetFunctionLister() extensionslistersv1alpha1.FunctionLister {
	return extensionslistersv1alpha1.NewFunctionLister(l.IndexerFor(&extensionsv1alpha1.Function{}))
}
