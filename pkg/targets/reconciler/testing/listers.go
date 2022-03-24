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

package testing

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	rbaclistersv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"

	rt "knative.dev/pkg/reconciler/testing"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	fakeservingclient "knative.dev/serving/pkg/client/clientset/versioned/fake"
	servinglistersv1 "knative.dev/serving/pkg/client/listers/serving/v1"

	targetsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	faketargetsclient "github.com/triggermesh/triggermesh/pkg/client/generated/clientset/internalclientset/fake"
	targetslisters "github.com/triggermesh/triggermesh/pkg/client/generated/listers/targets/v1alpha1"
)

var clientSetSchemes = []func(*runtime.Scheme) error{
	faketargetsclient.AddToScheme,
	fakek8sclient.AddToScheme,
	fakeservingclient.AddToScheme,
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

// GetTargetsObjects returns objects from the targets API.
func (l *Listers) GetTargetsObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(faketargetsclient.AddToScheme)
}

// GetKubeObjects returns objects from Kubernetes APIs.
func (l *Listers) GetKubeObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakek8sclient.AddToScheme)
}

// GetServingObjects returns objects from the serving API.
func (l *Listers) GetServingObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakeservingclient.AddToScheme)
}

// GetServiceAccountLister returns a lister for ServiceAccount objects.
func (l *Listers) GetServiceAccountLister() corelistersv1.ServiceAccountLister {
	return corelistersv1.NewServiceAccountLister(l.IndexerFor(&corev1.ServiceAccount{}))
}

// GetRoleBindingLister returns a lister for RoleBinding objects
func (l *Listers) GetRoleBindingLister() rbaclistersv1.RoleBindingLister {
	return rbaclistersv1.NewRoleBindingLister(l.IndexerFor(&rbacv1.RoleBinding{}))
}

// GetAlibabaOSSTargetLister returns a Lister for AlibabaOSSTarget objects.
func (l *Listers) GetAlibabaOSSTargetLister() targetslisters.AlibabaOSSTargetLister {
	return targetslisters.NewAlibabaOSSTargetLister(l.IndexerFor(&targetsv1alpha1.AlibabaOSSTarget{}))
}

// GetAWSComprehendTargetLister returns a Lister for AWSComprehendTarget objects.
func (l *Listers) GetAWSComprehendTargetLister() targetslisters.AWSComprehendTargetLister {
	return targetslisters.NewAWSComprehendTargetLister(l.IndexerFor(&targetsv1alpha1.AWSComprehendTarget{}))
}

// GetAWSDynamoDBTargetLister returns a Lister for AWSDynamoDBTarget objects.
func (l *Listers) GetAWSDynamoDBTargetLister() targetslisters.AWSDynamoDBTargetLister {
	return targetslisters.NewAWSDynamoDBTargetLister(l.IndexerFor(&targetsv1alpha1.AWSDynamoDBTarget{}))
}

// GetAWSEventBridgeTargetLister returns a Lister for AWSEventBridgeTarget objects.
func (l *Listers) GetAWSEventBridgeTargetLister() targetslisters.AWSEventBridgeTargetLister {
	return targetslisters.NewAWSEventBridgeTargetLister(l.IndexerFor(&targetsv1alpha1.AWSEventBridgeTarget{}))
}

// GetAWSKinesisTargetLister returns a Lister for AWSKinesisTarget objects.
func (l *Listers) GetAWSKinesisTargetLister() targetslisters.AWSKinesisTargetLister {
	return targetslisters.NewAWSKinesisTargetLister(l.IndexerFor(&targetsv1alpha1.AWSKinesisTarget{}))
}

// GetAWSLambdaTargetLister returns a Lister for AWSLambdaTarget objects.
func (l *Listers) GetAWSLambdaTargetLister() targetslisters.AWSLambdaTargetLister {
	return targetslisters.NewAWSLambdaTargetLister(l.IndexerFor(&targetsv1alpha1.AWSLambdaTarget{}))
}

// GetAWSS3TargetLister returns a Lister for AWSS3Target objects.
func (l *Listers) GetAWSS3TargetLister() targetslisters.AWSS3TargetLister {
	return targetslisters.NewAWSS3TargetLister(l.IndexerFor(&targetsv1alpha1.AWSS3Target{}))
}

// GetAWSSNSTargetLister returns a Lister for AWSSNSTarget objects.
func (l *Listers) GetAWSSNSTargetLister() targetslisters.AWSSNSTargetLister {
	return targetslisters.NewAWSSNSTargetLister(l.IndexerFor(&targetsv1alpha1.AWSSNSTarget{}))
}

// GetAWSSQSTargetLister returns a Lister for AWSSQSTarget objects.
func (l *Listers) GetAWSSQSTargetLister() targetslisters.AWSSQSTargetLister {
	return targetslisters.NewAWSSQSTargetLister(l.IndexerFor(&targetsv1alpha1.AWSSQSTarget{}))
}

// GetAzureEventHubsTargetLister returns a Lister for AzureEventHubsTarget objects.
func (l *Listers) GetAzureEventHubsTargetLister() targetslisters.AzureEventHubsTargetLister {
	return targetslisters.NewAzureEventHubsTargetLister(l.IndexerFor(&targetsv1alpha1.AzureEventHubsTarget{}))
}

// GetConfluentTargetLister returns a Lister for ConfluentTarget objects.
func (l *Listers) GetConfluentTargetLister() targetslisters.ConfluentTargetLister {
	return targetslisters.NewConfluentTargetLister(l.IndexerFor(&targetsv1alpha1.ConfluentTarget{}))
}

// GetDatadogTargetLister returns a Lister for DatadogTarget objects.
func (l *Listers) GetDatadogTargetLister() targetslisters.DatadogTargetLister {
	return targetslisters.NewDatadogTargetLister(l.IndexerFor(&targetsv1alpha1.DatadogTarget{}))
}

// GetElasticsearchTargetLister returns a Lister for ElasticsearchTarget objects.
func (l *Listers) GetElasticsearchTargetLister() targetslisters.ElasticsearchTargetLister {
	return targetslisters.NewElasticsearchTargetLister(l.IndexerFor(&targetsv1alpha1.ElasticsearchTarget{}))
}

// GetGoogleCloudFirestoreTargetLister returns a Lister for GoogleCloudFirestoreTarget objects.
func (l *Listers) GetGoogleCloudFirestoreTargetLister() targetslisters.GoogleCloudFirestoreTargetLister {
	return targetslisters.NewGoogleCloudFirestoreTargetLister(l.IndexerFor(&targetsv1alpha1.GoogleCloudFirestoreTarget{}))
}

// GetGoogleCloudStorageTargetLister returns a Lister for GoogleCloudStorageTarget objects.
func (l *Listers) GetGoogleCloudStorageTargetLister() targetslisters.GoogleCloudStorageTargetLister {
	return targetslisters.NewGoogleCloudStorageTargetLister(l.IndexerFor(&targetsv1alpha1.GoogleCloudStorageTarget{}))
}

// GetGoogleCloudWorkflowsTargetLister returns a Lister for GoogleCloudWorkflowsTarget objects.
func (l *Listers) GetGoogleCloudWorkflowsTargetLister() targetslisters.GoogleCloudWorkflowsTargetLister {
	return targetslisters.NewGoogleCloudWorkflowsTargetLister(l.IndexerFor(&targetsv1alpha1.GoogleCloudWorkflowsTarget{}))
}

// GetGoogleSheetTargetLister returns a Lister for GoogleSheetTarget objects.
func (l *Listers) GetGoogleSheetTargetLister() targetslisters.GoogleSheetTargetLister {
	return targetslisters.NewGoogleSheetTargetLister(l.IndexerFor(&targetsv1alpha1.GoogleSheetTarget{}))
}

// GetHasuraTargetLister returns a Lister for HasuraTarget objects.
func (l *Listers) GetHasuraTargetLister() targetslisters.HasuraTargetLister {
	return targetslisters.NewHasuraTargetLister(l.IndexerFor(&targetsv1alpha1.HasuraTarget{}))
}

// GetHTTPTargetLister returns a Lister for HTTPTarget objects.
func (l *Listers) GetHTTPTargetLister() targetslisters.HTTPTargetLister {
	return targetslisters.NewHTTPTargetLister(l.IndexerFor(&targetsv1alpha1.HTTPTarget{}))
}

// GetIBMMQTargetLister returns a Lister for IBMMQTarget objects.
func (l *Listers) GetIBMMQTargetLister() targetslisters.IBMMQTargetLister {
	return targetslisters.NewIBMMQTargetLister(l.IndexerFor(&targetsv1alpha1.IBMMQTarget{}))
}

// GetInfraTargetLister returns a Lister for InfraTarget objects.
func (l *Listers) GetInfraTargetLister() targetslisters.InfraTargetLister {
	return targetslisters.NewInfraTargetLister(l.IndexerFor(&targetsv1alpha1.InfraTarget{}))
}

// GetJiraTargetLister returns a Lister for JiraTarget objects.
func (l *Listers) GetJiraTargetLister() targetslisters.JiraTargetLister {
	return targetslisters.NewJiraTargetLister(l.IndexerFor(&targetsv1alpha1.JiraTarget{}))
}

// GetLogzMetricsTargetLister returns a Lister for LogzMetricsTarget objects.
func (l *Listers) GetLogzMetricsTargetLister() targetslisters.LogzMetricsTargetLister {
	return targetslisters.NewLogzMetricsTargetLister(l.IndexerFor(&targetsv1alpha1.LogzMetricsTarget{}))
}

// GetLogzTargetLister returns a Lister for LogzTarget objects.
func (l *Listers) GetLogzTargetLister() targetslisters.LogzTargetLister {
	return targetslisters.NewLogzTargetLister(l.IndexerFor(&targetsv1alpha1.LogzTarget{}))
}

// GetOracleTargetLister returns a Lister for OracleTarget objects.
func (l *Listers) GetOracleTargetLister() targetslisters.OracleTargetLister {
	return targetslisters.NewOracleTargetLister(l.IndexerFor(&targetsv1alpha1.OracleTarget{}))
}

// GetSalesforceTargetLister returns a Lister for SalesforceTarget objects.
func (l *Listers) GetSalesforceTargetLister() targetslisters.SalesforceTargetLister {
	return targetslisters.NewSalesforceTargetLister(l.IndexerFor(&targetsv1alpha1.SalesforceTarget{}))
}

// GetSendGridTargetLister returns a Lister for SendGridTarget objects.
func (l *Listers) GetSendGridTargetLister() targetslisters.SendGridTargetLister {
	return targetslisters.NewSendGridTargetLister(l.IndexerFor(&targetsv1alpha1.SendGridTarget{}))
}

// GetSlackTargetLister returns a Lister for SlackTarget objects.
func (l *Listers) GetSlackTargetLister() targetslisters.SlackTargetLister {
	return targetslisters.NewSlackTargetLister(l.IndexerFor(&targetsv1alpha1.SlackTarget{}))
}

// GetSplunkTargetLister returns a Lister for SplunkTarget objects.
func (l *Listers) GetSplunkTargetLister() targetslisters.SplunkTargetLister {
	return targetslisters.NewSplunkTargetLister(l.IndexerFor(&targetsv1alpha1.SplunkTarget{}))
}

// GetServiceLister returns a lister for Service objects.
func (l *Listers) GetServiceLister() servinglistersv1.ServiceLister {
	return servinglistersv1.NewServiceLister(l.IndexerFor(&servingv1.Service{}))
}
