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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// InfraTargetLister helps list InfraTargets.
// All objects returned here must be treated as read-only.
type InfraTargetLister interface {
	// List lists all InfraTargets in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.InfraTarget, err error)
	// InfraTargets returns an object that can list and get InfraTargets.
	InfraTargets(namespace string) InfraTargetNamespaceLister
	InfraTargetListerExpansion
}

// infraTargetLister implements the InfraTargetLister interface.
type infraTargetLister struct {
	indexer cache.Indexer
}

// NewInfraTargetLister returns a new InfraTargetLister.
func NewInfraTargetLister(indexer cache.Indexer) InfraTargetLister {
	return &infraTargetLister{indexer: indexer}
}

// List lists all InfraTargets in the indexer.
func (s *infraTargetLister) List(selector labels.Selector) (ret []*v1alpha1.InfraTarget, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.InfraTarget))
	})
	return ret, err
}

// InfraTargets returns an object that can list and get InfraTargets.
func (s *infraTargetLister) InfraTargets(namespace string) InfraTargetNamespaceLister {
	return infraTargetNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// InfraTargetNamespaceLister helps list and get InfraTargets.
// All objects returned here must be treated as read-only.
type InfraTargetNamespaceLister interface {
	// List lists all InfraTargets in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.InfraTarget, err error)
	// Get retrieves the InfraTarget from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.InfraTarget, error)
	InfraTargetNamespaceListerExpansion
}

// infraTargetNamespaceLister implements the InfraTargetNamespaceLister
// interface.
type infraTargetNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all InfraTargets in the indexer for a given namespace.
func (s infraTargetNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.InfraTarget, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.InfraTarget))
	})
	return ret, err
}

// Get retrieves the InfraTarget from the indexer for a given namespace and name.
func (s infraTargetNamespaceLister) Get(name string) (*v1alpha1.InfraTarget, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("infratarget"), name)
	}
	return obj.(*v1alpha1.InfraTarget), nil
}
