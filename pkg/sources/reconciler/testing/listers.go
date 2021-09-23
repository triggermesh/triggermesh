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

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	fakeclient "github.com/triggermesh/triggermesh/pkg/client/generated/clientset/internalclientset/fake"
	listersv1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/sources/v1alpha1"
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

// GetSourcesObjects returns objects from the sources API.
func (l *Listers) GetSourcesObjects() []runtime.Object {
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

// GetAWSCloudWatchSourceLister returns a Lister for AWSCloudWatchSource objects.
func (l *Listers) GetAWSCloudWatchSourceLister() listersv1alpha1.AWSCloudWatchSourceLister {
	return listersv1alpha1.NewAWSCloudWatchSourceLister(l.IndexerFor(&v1alpha1.AWSCloudWatchSource{}))
}

// GetAWSCloudWatchLogsSourceLister returns a Lister for AWSCloudWatchSource objects.
func (l *Listers) GetAWSCloudWatchLogsSourceLister() listersv1alpha1.AWSCloudWatchLogsSourceLister {
	return listersv1alpha1.NewAWSCloudWatchLogsSourceLister(l.IndexerFor(&v1alpha1.AWSCloudWatchLogsSource{}))
}

// GetAWSCodeCommitSourceLister returns a Lister for AWSCodeCommitSource objects.
func (l *Listers) GetAWSCodeCommitSourceLister() listersv1alpha1.AWSCodeCommitSourceLister {
	return listersv1alpha1.NewAWSCodeCommitSourceLister(l.IndexerFor(&v1alpha1.AWSCodeCommitSource{}))
}

// GetAWSCognitoUserPoolSourceLister returns a Lister for AWSCognitoIdentitySource objects.
func (l *Listers) GetAWSCognitoUserPoolSourceLister() listersv1alpha1.AWSCognitoUserPoolSourceLister {
	return listersv1alpha1.NewAWSCognitoUserPoolSourceLister(l.IndexerFor(&v1alpha1.AWSCognitoUserPoolSource{}))
}

// GetAWSCognitoIdentitySourceLister returns a Lister for AWSCognitoUserPoolSource objects.
func (l *Listers) GetAWSCognitoIdentitySourceLister() listersv1alpha1.AWSCognitoIdentitySourceLister {
	return listersv1alpha1.NewAWSCognitoIdentitySourceLister(l.IndexerFor(&v1alpha1.AWSCognitoIdentitySource{}))
}

// GetAWSDynamoDBSourceLister returns a Lister for AWSDynamoDBSource objects.
func (l *Listers) GetAWSDynamoDBSourceLister() listersv1alpha1.AWSDynamoDBSourceLister {
	return listersv1alpha1.NewAWSDynamoDBSourceLister(l.IndexerFor(&v1alpha1.AWSDynamoDBSource{}))
}

// GetAWSKinesisSourceLister returns a Lister for AWSKinesisSource objects.
func (l *Listers) GetAWSKinesisSourceLister() listersv1alpha1.AWSKinesisSourceLister {
	return listersv1alpha1.NewAWSKinesisSourceLister(l.IndexerFor(&v1alpha1.AWSKinesisSource{}))
}

// GetAWSSNSSourceLister returns a Lister for AWSSNSSource objects.
func (l *Listers) GetAWSSNSSourceLister() listersv1alpha1.AWSSNSSourceLister {
	return listersv1alpha1.NewAWSSNSSourceLister(l.IndexerFor(&v1alpha1.AWSSNSSource{}))
}

// GetAWSSQSSourceLister returns a Lister for AWSSQSSource objects.
func (l *Listers) GetAWSSQSSourceLister() listersv1alpha1.AWSSQSSourceLister {
	return listersv1alpha1.NewAWSSQSSourceLister(l.IndexerFor(&v1alpha1.AWSSQSSource{}))
}

// GetHTTPPollerSourceLister returns a Lister for HTTPPollerSource objects.
func (l *Listers) GetHTTPPollerSourceLister() listersv1alpha1.HTTPPollerSourceLister {
	return listersv1alpha1.NewHTTPPollerSourceLister(l.IndexerFor(&v1alpha1.HTTPPollerSource{}))
}

// GetSlackSourceLister returns a Lister for SlackSource objects.
func (l *Listers) GetSlackSourceLister() listersv1alpha1.SlackSourceLister {
	return listersv1alpha1.NewSlackSourceLister(l.IndexerFor(&v1alpha1.SlackSource{}))
}

// GetWebhookSourceLister returns a Lister for WebhookSource objects.
func (l *Listers) GetWebhookSourceLister() listersv1alpha1.WebhookSourceLister {
	return listersv1alpha1.NewWebhookSourceLister(l.IndexerFor(&v1alpha1.WebhookSource{}))
}

// GetZendeskSourceLister returns a Lister for ZendeskSource objects.
func (l *Listers) GetZendeskSourceLister() listersv1alpha1.ZendeskSourceLister {
	return listersv1alpha1.NewZendeskSourceLister(l.IndexerFor(&v1alpha1.ZendeskSource{}))
}

// GetDeploymentLister returns a lister for Deployment objects.
func (l *Listers) GetDeploymentLister() appslistersv1.DeploymentLister {
	return appslistersv1.NewDeploymentLister(l.IndexerFor(&appsv1.Deployment{}))
}

// GetServiceLister returns a lister for Knative Service objects.
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
