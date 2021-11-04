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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	targetsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	internalclientset "github.com/triggermesh/triggermesh/pkg/client/generated/clientset/internalclientset"
	internalinterfaces "github.com/triggermesh/triggermesh/pkg/client/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/triggermesh/triggermesh/pkg/client/generated/listers/targets/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// AzureEventHubsTargetInformer provides access to a shared informer and lister for
// AzureEventHubsTargets.
type AzureEventHubsTargetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.AzureEventHubsTargetLister
}

type azureEventHubsTargetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewAzureEventHubsTargetInformer constructs a new informer for AzureEventHubsTarget type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewAzureEventHubsTargetInformer(client internalclientset.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredAzureEventHubsTargetInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredAzureEventHubsTargetInformer constructs a new informer for AzureEventHubsTarget type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredAzureEventHubsTargetInformer(client internalclientset.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.TargetsV1alpha1().AzureEventHubsTargets(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.TargetsV1alpha1().AzureEventHubsTargets(namespace).Watch(context.TODO(), options)
			},
		},
		&targetsv1alpha1.AzureEventHubsTarget{},
		resyncPeriod,
		indexers,
	)
}

func (f *azureEventHubsTargetInformer) defaultInformer(client internalclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredAzureEventHubsTargetInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *azureEventHubsTargetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&targetsv1alpha1.AzureEventHubsTarget{}, f.defaultInformer)
}

func (f *azureEventHubsTargetInformer) Lister() v1alpha1.AzureEventHubsTargetLister {
	return v1alpha1.NewAzureEventHubsTargetLister(f.Informer().GetIndexer())
}
