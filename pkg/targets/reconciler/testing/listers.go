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

// GetGoogleSheetTargetsObjects returns objects from the targets API.
func (l *Listers) GetGoogleSheetTargetsObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(faketargetsclient.AddToScheme)
}

// GetSplunkTargetsObjects returns objects from the targets API.
func (l *Listers) GetSplunkTargetsObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(faketargetsclient.AddToScheme)
}

// GetHTTPTargetsObjects returns objects from the targets API.
func (l *Listers) GetHTTPTargetsObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(faketargetsclient.AddToScheme)
}

// GetHasuraTargetsObjects returns objects from the targets API.
func (l *Listers) GetHasuraTargetsObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(faketargetsclient.AddToScheme)
}

// GetLogzMetricsTargetObjects returns objects from the targets API.
func (l *Listers) GetLogzMetricsTargetObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(faketargetsclient.AddToScheme)
}

// GetLogzTargetObjects returns objects from the targets API.
func (l *Listers) GetLogzTargetObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(faketargetsclient.AddToScheme)
}

// GetAlibabaOSSTargetLister returns a Lister for AlibabaOSSTarget objects.
func (l *Listers) GetAlibabaOSSTargetLister() targetslisters.AlibabaOSSTargetLister {
	return targetslisters.NewAlibabaOSSTargetLister(l.IndexerFor(&targetsv1alpha1.AlibabaOSSTarget{}))
}

// GetAWSComprehendTargetLister returns a Lister for AWSComprehendTarget objects.
func (l *Listers) GetAWSComprehendTargetLister() targetslisters.AWSComprehendTargetLister {
	return targetslisters.NewAWSComprehendTargetLister(l.IndexerFor(&targetsv1alpha1.AWSComprehendTarget{}))
}

// GetGoogleSheetTargetLister returns a Lister for GoogleSheetTarget objects.
func (l *Listers) GetGoogleSheetTargetLister() targetslisters.GoogleSheetTargetLister {
	return targetslisters.NewGoogleSheetTargetLister(l.IndexerFor(&targetsv1alpha1.GoogleSheetTarget{}))
}

// GetSplunkTargetLister returns a Lister for SplunkTarget objects.
func (l *Listers) GetSplunkTargetLister() targetslisters.SplunkTargetLister {
	return targetslisters.NewSplunkTargetLister(l.IndexerFor(&targetsv1alpha1.SplunkTarget{}))
}

// GetHTTPTargetLister returns a Lister for HTTPTarget objects.
func (l *Listers) GetHTTPTargetLister() targetslisters.HTTPTargetLister {
	return targetslisters.NewHTTPTargetLister(l.IndexerFor(&targetsv1alpha1.HTTPTarget{}))
}

// GetHasuraTargetLister returns a Lister for HasuraTarget objects.
func (l *Listers) GetHasuraTargetLister() targetslisters.HasuraTargetLister {
	return targetslisters.NewHasuraTargetLister(l.IndexerFor(&targetsv1alpha1.HasuraTarget{}))
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

// GetSalesforceTargetLister returns a Lister for SalesforceTarget objects.
func (l *Listers) GetSalesforceTargetLister() targetslisters.SalesforceTargetLister {
	return targetslisters.NewSalesforceTargetLister(l.IndexerFor(&targetsv1alpha1.SalesforceTarget{}))
}

// GetServiceLister returns a lister for Service objects.
func (l *Listers) GetServiceLister() servinglistersv1.ServiceLister {
	return servinglistersv1.NewServiceLister(l.IndexerFor(&servingv1.Service{}))
}
