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
	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// TwilioSourceLister helps list TwilioSources.
// All objects returned here must be treated as read-only.
type TwilioSourceLister interface {
	// List lists all TwilioSources in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.TwilioSource, err error)
	// TwilioSources returns an object that can list and get TwilioSources.
	TwilioSources(namespace string) TwilioSourceNamespaceLister
	TwilioSourceListerExpansion
}

// twilioSourceLister implements the TwilioSourceLister interface.
type twilioSourceLister struct {
	indexer cache.Indexer
}

// NewTwilioSourceLister returns a new TwilioSourceLister.
func NewTwilioSourceLister(indexer cache.Indexer) TwilioSourceLister {
	return &twilioSourceLister{indexer: indexer}
}

// List lists all TwilioSources in the indexer.
func (s *twilioSourceLister) List(selector labels.Selector) (ret []*v1alpha1.TwilioSource, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.TwilioSource))
	})
	return ret, err
}

// TwilioSources returns an object that can list and get TwilioSources.
func (s *twilioSourceLister) TwilioSources(namespace string) TwilioSourceNamespaceLister {
	return twilioSourceNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// TwilioSourceNamespaceLister helps list and get TwilioSources.
// All objects returned here must be treated as read-only.
type TwilioSourceNamespaceLister interface {
	// List lists all TwilioSources in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.TwilioSource, err error)
	// Get retrieves the TwilioSource from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.TwilioSource, error)
	TwilioSourceNamespaceListerExpansion
}

// twilioSourceNamespaceLister implements the TwilioSourceNamespaceLister
// interface.
type twilioSourceNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all TwilioSources in the indexer for a given namespace.
func (s twilioSourceNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.TwilioSource, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.TwilioSource))
	})
	return ret, err
}

// Get retrieves the TwilioSource from the indexer for a given namespace and name.
func (s twilioSourceNamespaceLister) Get(name string) (*v1alpha1.TwilioSource, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("twiliosource"), name)
	}
	return obj.(*v1alpha1.TwilioSource), nil
}
