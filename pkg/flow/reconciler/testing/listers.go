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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	rbaclistersv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"

	fakeeventingclientset "knative.dev/eventing/pkg/client/clientset/versioned/fake"
	rt "knative.dev/pkg/reconciler/testing"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	fakeservingclient "knative.dev/serving/pkg/client/clientset/versioned/fake"
	servinglistersv1 "knative.dev/serving/pkg/client/listers/serving/v1"

	flowv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	fakeflowclient "github.com/triggermesh/triggermesh/pkg/client/generated/clientset/internalclientset/fake"
	flowlisters "github.com/triggermesh/triggermesh/pkg/client/generated/listers/flow/v1alpha1"
)

var clientSetSchemes = []func(*runtime.Scheme) error{
	fakeflowclient.AddToScheme,
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

// GetDataWeaveTransformationLister returns a Lister for DataWeaveTransformation objects.
func (l *Listers) GetDataWeaveTransformationLister() flowlisters.DataWeaveTransformationLister {
	return flowlisters.NewDataWeaveTransformationLister(l.IndexerFor(&flowv1alpha1.DataWeaveTransformation{}))
}

// GetFlowObjects returns objects from the flow API.
func (l *Listers) GetFlowObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakeflowclient.AddToScheme)
}

// GetKubeObjects returns objects from Kubernetes APIs.
func (l *Listers) GetKubeObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakek8sclient.AddToScheme)
}

// GetServingObjects returns objects from the serving API.
func (l *Listers) GetServingObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakeservingclient.AddToScheme)
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
func (l *Listers) GetJQTransformationLister() flowlisters.JQTransformationLister {
	return flowlisters.NewJQTransformationLister(l.IndexerFor(&flowv1alpha1.JQTransformation{}))
}

// GetSynchronizerLister returns a Lister for Synchronizer objects.
func (l *Listers) GetSynchronizerLister() flowlisters.SynchronizerLister {
	return flowlisters.NewSynchronizerLister(l.IndexerFor(&flowv1alpha1.Synchronizer{}))
}

// GetTransformationLister returns a Lister for Transformation objects.
func (l *Listers) GetTransformationLister() flowlisters.TransformationLister {
	return flowlisters.NewTransformationLister(l.IndexerFor(&flowv1alpha1.Transformation{}))
}

// GetXMLToJSONTransformationLister returns a Lister for XMLToJSONTransformation objects.
func (l *Listers) GetXMLToJSONTransformationLister() flowlisters.XMLToJSONTransformationLister {
	return flowlisters.NewXMLToJSONTransformationLister(l.IndexerFor(&flowv1alpha1.XMLToJSONTransformation{}))
}

// GetXSLTTransformationLister returns a Lister for XSLTTransformation objects.
func (l *Listers) GetXSLTTransformationLister() flowlisters.XSLTTransformationLister {
	return flowlisters.NewXSLTTransformationLister(l.IndexerFor(&flowv1alpha1.XSLTTransformation{}))
}
