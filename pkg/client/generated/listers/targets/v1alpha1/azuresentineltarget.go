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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// AzureSentinelTargetLister helps list AzureSentinelTargets.
// All objects returned here must be treated as read-only.
type AzureSentinelTargetLister interface {
	// List lists all AzureSentinelTargets in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.AzureSentinelTarget, err error)
	// AzureSentinelTargets returns an object that can list and get AzureSentinelTargets.
	AzureSentinelTargets(namespace string) AzureSentinelTargetNamespaceLister
	AzureSentinelTargetListerExpansion
}

// azureSentinelTargetLister implements the AzureSentinelTargetLister interface.
type azureSentinelTargetLister struct {
	indexer cache.Indexer
}

// NewAzureSentinelTargetLister returns a new AzureSentinelTargetLister.
func NewAzureSentinelTargetLister(indexer cache.Indexer) AzureSentinelTargetLister {
	return &azureSentinelTargetLister{indexer: indexer}
}

// List lists all AzureSentinelTargets in the indexer.
func (s *azureSentinelTargetLister) List(selector labels.Selector) (ret []*v1alpha1.AzureSentinelTarget, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AzureSentinelTarget))
	})
	return ret, err
}

// AzureSentinelTargets returns an object that can list and get AzureSentinelTargets.
func (s *azureSentinelTargetLister) AzureSentinelTargets(namespace string) AzureSentinelTargetNamespaceLister {
	return azureSentinelTargetNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// AzureSentinelTargetNamespaceLister helps list and get AzureSentinelTargets.
// All objects returned here must be treated as read-only.
type AzureSentinelTargetNamespaceLister interface {
	// List lists all AzureSentinelTargets in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.AzureSentinelTarget, err error)
	// Get retrieves the AzureSentinelTarget from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.AzureSentinelTarget, error)
	AzureSentinelTargetNamespaceListerExpansion
}

// azureSentinelTargetNamespaceLister implements the AzureSentinelTargetNamespaceLister
// interface.
type azureSentinelTargetNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all AzureSentinelTargets in the indexer for a given namespace.
func (s azureSentinelTargetNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.AzureSentinelTarget, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AzureSentinelTarget))
	})
	return ret, err
}

// Get retrieves the AzureSentinelTarget from the indexer for a given namespace and name.
func (s azureSentinelTargetNamespaceLister) Get(name string) (*v1alpha1.AzureSentinelTarget, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("azuresentineltarget"), name)
	}
	return obj.(*v1alpha1.AzureSentinelTarget), nil
}
